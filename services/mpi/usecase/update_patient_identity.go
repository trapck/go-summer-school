package usecase

import (
	"context"
	"fmt"
	"net/http"
	"time"

	uuid "github.com/satori/go.uuid"

	"wasfaty.api/pkg/cerror"
	"wasfaty.api/pkg/converto"
	fhirModel "wasfaty.api/pkg/fhir/model"
	"wasfaty.api/services/mpi/entity"
)

func (uc *UseCase) UpdatePatientIdentity(ctx context.Context, id fhirModel.ID, p *fhirModel.Parameters) (
	*fhirModel.Task, error) {
	if _, err := uc.fhir.ValidateParameters(ctx, p); err != nil {
		return nil, err
	}

	params, err := uc.extractUpdatePatientIdentityParams(ctx, p)
	if err != nil {
		return nil, err
	}

	if pErr := uc.validatePatientProfile(ctx, p.Meta, params.Patient.Meta); pErr != nil {
		return nil, pErr
	}

	if params.Patient.ID != id {
		return nil, cerror.NewF(ctx, cerror.KindBadValidation, "url id and patient id are not equal").LogError()
	}

	dbPatient, err := uc.fhir.GetPatientByID(ctx, id)
	if err != nil {
		return nil, err
	}

	err = uc.validateUpdatePatientIdentityByInternalRules(ctx, params.Patient, dbPatient)
	if err != nil {
		return nil, err
	}

	err = uc.validatePatientDupls(ctx, params.Patient)
	if err != nil {
		return nil, err
	}

	duplTasks, err := uc.searchUpdatePatientIdentifierDuplicateTasks(ctx, params.Patient)
	if err != nil {
		return nil, err
	}

	t := prepareUpdatePatientIdentityTask(p)

	otp, err := uc.otp.GenerateByPhone(ctx, params.ConfirmationMethod, string(t.ID))
	if err != nil {
		return nil, err
	}

	err = uc.sendOTP(ctx, otp)
	if err != nil {
		return nil, err
	}

	_, err = uc.saveUpdatePatientIdentityBundle(ctx, p, t, duplTasks)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (uc *UseCase) validateUpdatePatientIdentityByInternalRules(
	ctx context.Context,
	patient *fhirModel.Patient,
	dbPatient *fhirModel.Patient) error {
	if !converto.BoolValue(dbPatient.Active) {
		return cerror.NewF(ctx, cerror.KindBadValidation, "patient has inactive status").LogError()
	}

	if converto.BoolValue(dbPatient.DeceasedBoolean) {
		return cerror.NewF(ctx, cerror.KindBadValidation, "patient has deceased status").LogError()
	}

	return uc.validateIdents(ctx, patient, 1)
}

func (uc *UseCase) searchUpdatePatientIdentifierDuplicateTasks(
	ctx context.Context,
	p *fhirModel.Patient) (
	[]*fhirModel.Task, error) {
	tasks, err := uc.searchTasksByIdent(ctx, p)
	if err != nil {
		return nil, err
	}

	duplIDs := make(map[fhirModel.ID]interface{})
	duplTasks := []*fhirModel.Task{}

	for _, t := range tasks {
		if _, ok := duplIDs[t.ID]; !ok && t.Status == fhirModel.TaskStatusInProgress {
			duplIDs[t.ID] = nil
			duplTasks = append(duplTasks, t)
		}
	}

	return duplTasks, nil
}

//nolint:dupl
func prepareUpdatePatientIdentityTask(p *fhirModel.Parameters) *fhirModel.Task {
	return &fhirModel.Task{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{
				ID:           fhirModel.ID(uuid.NewV4().String()),
				ResourceType: fhirModel.ResourceTask,
				Meta:         &fhirModel.Meta{Profile: []string{fhirModel.StructureDefinitionTaskPatientUpdateIdentity}},
			},
		},
		Status:         fhirModel.TaskStatusInProgress,
		BusinessStatus: &fhirModel.CodeableConcept{Text: entity.TaskBusinessStatusPatientIdentityUpdated},
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

func (uc *UseCase) extractUpdatePatientIdentityParams(ctx context.Context, p *fhirModel.Parameters) (
	*entity.UpdatePatientIdentityParameters, error) {
	var (
		confirmationMethod string
		patientParam       interface{}
	)

	for _, param := range p.Parameter {
		switch param.Name {
		case "confirmationMethod":
			confirmationMethod = converto.StringValue(param.ValueString)
		case "patient":
			patientParam = param.Resource
		}
	}

	if confirmationMethod == "" || patientParam == nil {
		return nil, cerror.NewF(
			ctx,
			cerror.KindBadValidation,
			"patient or confirmation method parameter is empty").LogError()
	}

	patient := new(fhirModel.Patient)
	if err := interfaceToStruct(ctx, patientParam, patient, "Parameters.parameter[1]"); err != nil {
		return nil, err
	}

	return &entity.UpdatePatientIdentityParameters{
		ConfirmationMethod: confirmationMethod,
		Patient:            patient,
	}, nil
}

//nolint:dupl
func (uc *UseCase) saveUpdatePatientIdentityBundle(
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
