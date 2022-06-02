package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	uuid "github.com/satori/go.uuid"
	"wasfaty.api/pkg/cerror"
	"wasfaty.api/pkg/converto"
	fhirModel "wasfaty.api/pkg/fhir/model"
	"wasfaty.api/pkg/log"
	"wasfaty.api/services/mpi/entity"
)

func (uc *UseCase) CreatePatient(ctx context.Context, params *fhirModel.Parameters) (
	*fhirModel.Task, error) {
	patient, err := uc.unmarshalPatientParam(ctx, params, 0)
	if err != nil {
		return nil, err
	}

	if vErr := uc.validateCreatePatientParameters(ctx, params, patient); vErr != nil {
		return nil, vErr
	}

	if pErr := uc.validatePatientProfile(ctx, params.Meta, patient.Meta); pErr != nil {
		return nil, pErr
	}

	err = uc.validateCreatePatientByInternalRules(ctx, patient)
	if err != nil {
		return nil, err
	}

	err = uc.validateByExtDocRegistry(ctx, patient)
	if err != nil {
		return nil, err
	}

	err = uc.validatePatientDupls(ctx, patient)
	if err != nil {
		return nil, err
	}

	duplTasks, err := uc.searchDuplicateTasks(ctx, patient)
	if err != nil {
		return nil, err
	}

	task := prepareCreatePatientTask(params)

	otp, err := uc.generateOTP(ctx, task.ID, patient)
	if err != nil {
		return nil, err
	}

	err = uc.sendOTP(ctx, otp)
	if err != nil {
		return nil, err
	}

	_, err = uc.saveCreatePatientBundle(ctx, params, task, duplTasks)
	if err != nil {
		return nil, err
	}

	return task, nil
}

// nolint:unparam
func (uc *UseCase) unmarshalPatientParam(ctx context.Context, p *fhirModel.Parameters, patientParamIndex int) (
	*fhirModel.Patient, error) {
	if len(p.Parameter) < patientParamIndex+1 {
		return nil, cerror.NewValidationError(
			ctx, map[string]string{
				"Parameters": fmt.Sprintf("expected to have at least %d elements", patientParamIndex+1),
			}).LogError()
	}

	patient := new(fhirModel.Patient)
	if err := interfaceToStruct(
		ctx,
		p.Parameter[0].Resource,
		patient,
		fmt.Sprintf("Parameters.parameter[%d].resource", patientParamIndex)); err != nil {
		return nil, err
	}

	return patient, nil
}

func (uc *UseCase) searchDuplicateTasks(ctx context.Context, p *fhirModel.Patient) (
	[]*fhirModel.Task, error) {
	t, err := uc.searchTasksByIdent(ctx, p)
	if err != nil {
		return nil, err
	}

	if len(t) > 0 {
		return t, nil
	}

	t, err = uc.searchTasksByTelecom(ctx, p)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (uc *UseCase) searchTasksByIdent(ctx context.Context, p *fhirModel.Patient) (
	[]*fhirModel.Task, error) {
	uc.sortIdentifiers(p.Identifier)

	for i, ident := range p.Identifier {
		if len(ident.Type.Codings) == 0 {
			return nil, cerror.NewValidationError(
				ctx, map[string]string{fmt.Sprintf("Identifier[%d].type", i): "empty coding"}).
				LogError()
		}

		tasks, err := uc.fhir.SearchTaskByParams(ctx, &entity.SearchTaskParams{
			Identifier: &entity.SearchTaskByIdentifierParams{
				Value: ident.Value,
				Type:  fmt.Sprintf("%s|%s", ident.Type.Codings[0].System, ident.Type.Codings[0].Code),
			},
		})

		if err != nil {
			return nil, err
		}

		activeTasks := []*fhirModel.Task{}

		for _, t := range tasks {
			if t.Status == fhirModel.TaskStatusInProgress {
				activeTasks = append(activeTasks, t)
			}
		}

		if len(activeTasks) > 0 {
			return activeTasks, nil
		}
	}

	return []*fhirModel.Task{}, nil
}

func (uc *UseCase) sortIdentifiers(idents []*fhirModel.Identifier) {
	identOrder := entity.CreatePatientIdentPriority()
	identOrderLengh := len(identOrder)

	var index = func(code string) int {
		for i := 0; i < identOrderLengh; i++ {
			if identOrder[i] == code {
				return i
			}
		}
		return -1
	}

	sort.Slice(idents, func(i, j int) bool {
		a := idents[i]
		b := idents[j]
		// if a has no codings array it should be at the end of the sroted slice
		if len(a.Type.Codings) == 0 {
			return false
		}

		aIndex := index(a.Type.Codings[0].Code)
		// if a is not in the document list it should be at the end of the sroted slice
		if aIndex == -1 {
			return false
		}
		// if b has no codings array it should be at the end of the sroted slice
		if len(b.Type.Codings) == 0 {
			return true
		}

		bIndex := index(b.Type.Codings[0].Code)
		// if b is not in the document list it should be at the end of the sroted slice
		if bIndex == -1 {
			return true
		}

		return aIndex < bIndex
	})
}

func (uc *UseCase) searchTasksByTelecom(ctx context.Context, p *fhirModel.Patient) (
	[]*fhirModel.Task, error) {
	for _, telecom := range p.Telecom {
		if telecom.System != fhirModel.TelecomSystemPhone || telecom.Use != fhirModel.TelecomUseMobile {
			continue
		}

		if p.BirthDate == nil {
			return nil, cerror.NewF(ctx, cerror.KindBadValidation, "expected patient to have a birthdate")
		}

		tasks, err := uc.fhir.SearchTaskByParams(ctx, &entity.SearchTaskParams{
			Telecom: &entity.SearchTaskByTelecomParams{
				Value:     telecom.Value,
				System:    telecom.System,
				Use:       telecom.Use,
				BirthDate: p.BirthDate.String(),
			},
		})

		if err != nil {
			return nil, err
		}

		activeTasks := []*fhirModel.Task{}

		for _, t := range tasks {
			if t.Status == fhirModel.TaskStatusInProgress {
				activeTasks = append(activeTasks, t)
			}
		}

		if len(activeTasks) > 0 {
			return activeTasks, nil
		}
	}

	return []*fhirModel.Task{}, nil
}

func (uc *UseCase) validateCreatePatientParameters(ctx context.Context, params *fhirModel.Parameters, patient *fhirModel.Patient) error {
	if _, err := uc.fhir.ValidateParameters(ctx, params); err != nil {
		return err
	}

	if len(params.Parameter) != 1 {
		return cerror.NewValidationError(ctx, map[string]string{"Parameters": "expected to have 1 element"}).LogError()
	}

	if !converto.BoolValue(patient.Active) {
		return cerror.NewValidationError(
			ctx, map[string]string{"Parameters.parameter[0].resource.active": "should have true value"}).LogError()
	}

	paramsMap, err := interfaceToMap(ctx, params.Parameter[0].Resource, "Parameters.parameter[0].resource")
	if err != nil {
		return err
	}

	var forbiddenParams []string

	for _, fp := range entity.CreatePatientForbiddenParams() {
		if _, ok := paramsMap[fp]; ok {
			forbiddenParams = append(forbiddenParams, fp)
		}
	}

	if len(forbiddenParams) > 0 {
		errMsg := fmt.Sprintf("forbidden parameters: %s", strings.Join(forbiddenParams, ","))
		return cerror.NewValidationError(ctx, map[string]string{"Parameters.parameter[0].resource": errMsg}).LogError()
	}

	return nil
}

func (uc *UseCase) validateCreatePatientByInternalRules(ctx context.Context, patient *fhirModel.Patient) error {
	return uc.validateIdents(ctx, patient, 0)
}

func (uc *UseCase) validateIdents(ctx context.Context, p *fhirModel.Patient, patientParamIndex int) error {
	n := uc.getPatientNationalityCode(p)
	if n == "" {
		return cerror.NewValidationError(
			ctx, map[string]string{fmt.Sprintf("Parameters.parameter[%d]", patientParamIndex): "nationality is not passed"}).
			LogError()
	}

	for identIndex, i := range p.Identifier {
		if err := uc.validateIdentExpirationDate(ctx, i, patientParamIndex, identIndex); err != nil {
			return err
		}

		if err := uc.validateIdentValueFormat(ctx, i, patientParamIndex, identIndex); err != nil {
			return err
		}
	}

	if err := uc.validateRequiredIdentType(ctx, n, p.Identifier...); err != nil {
		return err
	}

	return nil
}

func (uc *UseCase) getPatientNationalityCode(p *fhirModel.Patient) string {
	for _, e := range p.Extension {
		if e.URL == fhirModel.StructureDefinitionPatientNationality {
			for _, ee := range e.Extension {
				if ee.URL == "code" && ee.ValueCodeableConcept != nil {
					for _, c := range ee.ValueCodeableConcept.Codings {
						return c.Code
					}
				}
			}
		}
	}

	return ""
}

//nolint:dupl
func (uc *UseCase) saveCreatePatientBundle(
	ctx context.Context,
	p *fhirModel.Parameters,
	t *fhirModel.Task,
	dupls []*fhirModel.Task) (*fhirModel.Bundle, error) {
	b := &fhirModel.Bundle{
		Resource: fhirModel.Resource{
			ID:           fhirModel.ID(uuid.NewV4().String()),
			ResourceType: fhirModel.ResourceBundle,
		},
		Type: fhirModel.BundleTypeTransaction,
		Entry: []*fhirModel.BundleEntry{
			{
				Resource: p,
				Request:  &fhirModel.BundleEntryRequest{Method: http.MethodPost, URL: fhirModel.ResourceParameters},
			},
			{
				Resource: t,
				Request:  &fhirModel.BundleEntryRequest{Method: http.MethodPost, URL: fhirModel.ResourceTask},
			},
		},
	}

	for _, d := range dupls {
		d.Status = fhirModel.TaskStatusCanceled
		b.Entry = append(b.Entry, &fhirModel.BundleEntry{
			Resource: d,
			Request: &fhirModel.BundleEntryRequest{
				Method: http.MethodPut,
				URL:    fmt.Sprintf("%s/%s", fhirModel.ResourceTask, d.ID),
			},
		})
	}

	return uc.fhir.CreateBundle(ctx, b)
}

//nolint:dupl
func prepareCreatePatientTask(p *fhirModel.Parameters) *fhirModel.Task {
	return &fhirModel.Task{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{
				ID:           fhirModel.ID(uuid.NewV4().String()),
				ResourceType: fhirModel.ResourceTask,
				Meta:         &fhirModel.Meta{Profile: []string{fhirModel.StructureDefinitionTaskPatientCreate}},
			},
		},

		Status:         fhirModel.TaskStatusInProgress,
		BusinessStatus: &fhirModel.CodeableConcept{Text: entity.TaskBusinessStatusOTPCodeSent},
		Intent:         fhirModel.TaskIntentOrder,
		AuthoredOn:     (*fhirModel.DateTime)(converto.TimePointer(time.Now().UTC())),
		Input: []*fhirModel.TaskInput{
			{
				Type: &fhirModel.CodeableConcept{
					Codings: []*fhirModel.Coding{
						{
							Code:   fhirModel.ResourceParameters,
							System: fhirModel.CodingSystemResourceTypes,
						},
					},
				},
				ValueX: fhirModel.ValueX{
					ValueReference: &fhirModel.Reference{
						Reference: fmt.Sprintf("%s/%s", fhirModel.ResourceParameters, p.ID),
					},
				},
			},
		},
	}
}

func (uc *UseCase) generateOTP(ctx context.Context, taskID fhirModel.ID, p *fhirModel.Patient) (
	*entity.OTP, error) {
	var phone string

	for _, t := range p.Telecom {
		if t.System == fhirModel.TelecomSystemPhone && t.Use == fhirModel.TelecomUseMobile {
			phone = t.Value
			break
		}
	}

	if phone == "" {
		return nil, cerror.NewValidationError(
			ctx, map[string]string{"Telecom": "no mobile phone found"}).LogError()
	}

	otp, err := uc.otp.GenerateByPhone(ctx, phone, string(taskID))
	if err != nil {
		return nil, err
	}

	return otp, nil
}

func (uc *UseCase) sendOTP(ctx context.Context, otp *entity.OTP) error {
	if otp.Code == "" {
		return cerror.NewF(ctx, cerror.KindInternal, "otp is empty").LogError()
	}

	log.InfoF(ctx, "generated otp %s for number %s", otp.Code, otp.Value)

	return nil
}

func (uc *UseCase) validateByExtDocRegistry(ctx context.Context, p *fhirModel.Patient) error {
	for i, ident := range p.Identifier {
		if len(ident.Type.Codings) == 0 {
			continue
		}

		if ident.Type.Codings[0].Code == fhirModel.IdentNationalID {
			r, err := uc.docReg.Search(ctx, ident)
			if err != nil {
				return err
			}

			if !r.IsValid {
				return cerror.NewValidationError(ctx, map[string]string{
					fmt.Sprintf("Identifiers[%d]", i): "document is not valid",
				}).LogError()
			}
		}
	}

	return nil
}

func (uc *UseCase) validateIdentExpirationDate(ctx context.Context, i *fhirModel.Identifier, paramIndex, identIndex int) error {
	if i.Period == nil || i.Period.End == nil {
		return nil
	}

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	if i.Period.End.Time().Before(today) {
		field := fmt.Sprintf("Parameters.parameter[%d].resource.identifier[%d].period.end", paramIndex, identIndex)
		return cerror.NewValidationError(ctx, map[string]string{field: "identifier is expired"}).LogError()
	}

	return nil
}

func (uc *UseCase) validateIdentValueFormat(ctx context.Context, i *fhirModel.Identifier, paramIndex, identIndex int) error {
	for _, c := range i.Type.Codings {
		if c.Code != fhirModel.IdentNationalID && c.Code != fhirModel.IdentPermanentResidentCardNumber {
			continue
		}

		if !uc.validateNationalIDValue(i.Value) {
			field := fmt.Sprintf("Parameters.parameter[%d].resource.identifier[%d].value", paramIndex, identIndex)
			return cerror.NewValidationError(ctx, map[string]string{field: "invalid identifier value format"}).LogError()
		}
	}

	return nil
}

func (uc *UseCase) validateRequiredIdentType(ctx context.Context, nationalityCode string, idents ...*fhirModel.Identifier) error {
	allowedCodes := entity.IdentifierCodeForSANationality()
	if nationalityCode != fhirModel.NationalityCodeSA {
		allowedCodes = entity.IdentifierCodeForOtherNationality()
	}

	for _, i := range idents {
		for _, c := range i.Type.Codings {
			if contains(allowedCodes, c.Code) {
				return nil
			}
		}
	}

	return cerror.NewF(ctx, cerror.KindBadValidation,
		"identifier code should be one of %s", strings.Join(allowedCodes, ",")).LogError()
}

func (uc *UseCase) validatePatientProfile(ctx context.Context, paramsMeta, patientMeta *fhirModel.Meta) error {
	if paramsMeta == nil {
		return cerror.NewValidationError(ctx, map[string]string{"Parameters.meta": "value is required"})
	}

	if len(paramsMeta.Profile) != 1 {
		return cerror.NewValidationError(ctx, map[string]string{"Parameters.meta.profile": "expected to have 1 value"})
	}

	if patientMeta == nil {
		return cerror.NewValidationError(ctx, map[string]string{"Resource.meta": "value is required"})
	}

	if len(patientMeta.Profile) != 1 {
		return cerror.NewValidationError(ctx, map[string]string{"Resource.meta.profile": "expected to have 1 value"})
	}

	paramProfile := paramsMeta.Profile[0]
	patientProfile := patientMeta.Profile[0]

	var expectedProfile string

	switch paramProfile {
	case fhirModel.StructureDefinitionPatientCreateRequest:
		expectedProfile = fhirModel.StructureDefinitionPatientIdentified
	case fhirModel.StructureDefinitionPatientUpdateRequest:
		expectedProfile = fhirModel.StructureDefinitionPatientOperationUpdate
	case fhirModel.StructureDefinitionPatientUpdateIdentityRequest:
		expectedProfile = fhirModel.StructureDefinitionPatientOperationUpdateIdentity
	case fhirModel.StructureDefinitionPatientUpdateEmailRequest:
		expectedProfile = fhirModel.StructureDefinitionPatientOperationUpdateEmail
	default:
		return cerror.NewValidationError(ctx, map[string]string{"Parameters.meta.profile[0]": "unknown value"})
	}

	if patientProfile != expectedProfile {
		return cerror.NewValidationError(
			ctx,
			map[string]string{"Resource.meta.profile[0]": "given patient profile is not supported for current operation"})
	}

	return nil
}

func (uc *UseCase) validateNationalIDValue(value string) bool {
	validLen := 10
	value = strings.Replace(value, " ", "", -1)

	if _, err := strconv.Atoi(value); err != nil {
		return false
	}

	if len(value) != validLen {
		return false
	}

	idtype, _ := strconv.Atoi(value[0:1])

	if idtype != 1 && idtype != 2 {
		return false
	}

	idarr := make([]int, len(value))
	for c := 0; c < 10; c++ {
		idarr[c], _ = strconv.Atoi(value[c : c+1])
	}

	sum := 0

	for c := 0; c < validLen; c++ {
		if c%2 == 0 {
			dd := fmt.Sprintf("%02d", idarr[c]*2) //nolint:gomnd
			fvalue, _ := strconv.Atoi(dd[0:1])
			svalue, _ := strconv.Atoi(dd[1:2])
			sum += fvalue + svalue
		} else {
			sum += idarr[c]
		}
	}

	return sum%10 == 0
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}

	return false
}

func mapToStruct(ctx context.Context, m map[string]interface{}, dst interface{}, param string) error {
	b, err := json.Marshal(m)
	if err != nil {
		return cerror.NewValidationError(ctx, map[string]string{param: err.Error()}).LogError()
	}

	err = json.Unmarshal(b, dst)
	if err != nil {
		return cerror.NewValidationError(ctx, map[string]string{param: fmt.Sprintf("%T. %s", dst, err.Error())}).
			LogError()
	}

	return nil
}

func interfaceToStruct(ctx context.Context, i, dst interface{}, param string) error {
	b, err := json.Marshal(i)
	if err != nil {
		return cerror.NewValidationError(ctx, map[string]string{param: err.Error()}).LogError()
	}

	err = json.Unmarshal(b, dst)
	if err != nil {
		return cerror.NewValidationError(ctx, map[string]string{param: fmt.Sprintf("%T. %s", dst, err.Error())}).
			LogError()
	}

	return nil
}

func interfaceToMap(ctx context.Context, i interface{}, param string) (map[string]interface{}, error) {
	if m, ok := i.(map[string]interface{}); ok {
		return m, nil
	}

	m := make(map[string]interface{})
	if err := interfaceToStruct(ctx, i, &m, param); err != nil {
		return nil, err
	}

	return m, nil
}

func mapFromStruct(src interface{}) map[string]interface{} {
	b, _ := json.Marshal(src)
	m := make(map[string]interface{})
	_ = json.Unmarshal(b, &m)

	return m
}
