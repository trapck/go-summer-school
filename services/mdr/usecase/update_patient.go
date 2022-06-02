package usecase

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	uuid "github.com/satori/go.uuid"
	"wasfaty.api/pkg/cerror"
	"wasfaty.api/pkg/converto"
	fhirModel "wasfaty.api/pkg/fhir/model"
	"wasfaty.api/services/mpi/entity"
)

func (uc *UseCase) UpdatePatient(ctx context.Context, id fhirModel.ID, params *fhirModel.Parameters) (*fhirModel.Task, error) {
	patient, err := uc.unmarshalPatientParam(ctx, params, 0)
	if err != nil {
		return nil, err
	}

	err = uc.validateParametersCount(ctx, patient)
	if err != nil {
		return nil, err
	}

	if pErr := uc.validatePatientProfile(ctx, params.Meta, patient.Meta); pErr != nil {
		return nil, pErr
	}

	if _, err = uc.fhir.ValidateParameters(ctx, params); err != nil {
		return nil, err
	}

	if patient.ID != id {
		return nil, cerror.NewF(ctx, cerror.KindBadValidation, "url id and patient id are not equal").LogError()
	}

	dbPatient, err := uc.fhir.GetPatientByID(ctx, id)
	if err != nil {
		return nil, err
	}

	err = uc.validatePatientByInternalRules(ctx, dbPatient)
	if err != nil {
		return nil, err
	}

	setPatientParams(dbPatient, patient)

	task := prepareUpdatePatientTask(params)

	b := prepareUpdatePatientBundle(params, task, dbPatient)

	_, err = uc.fhir.CreateBundle(ctx, b)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (uc *UseCase) validatePatientByInternalRules(
	ctx context.Context,
	dbPatient *fhirModel.Patient) error {
	if !converto.BoolValue(dbPatient.Active) {
		return cerror.NewF(ctx, cerror.KindBadValidation, "patient has inactive status").LogError()
	}

	if converto.BoolValue(dbPatient.DeceasedBoolean) {
		return cerror.NewF(ctx, cerror.KindBadValidation, "patient has deceased status").LogError()
	}

	return nil
}

func prepareUpdatePatientTask(p *fhirModel.Parameters) *fhirModel.Task {
	return &fhirModel.Task{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{
				ID:           fhirModel.ID(uuid.NewV4().String()),
				ResourceType: fhirModel.ResourceTask,
				Meta:         &fhirModel.Meta{Profile: []string{fhirModel.StructureDefinitionTaskPatientUpdate}},
			},
		},
		Status:     fhirModel.TaskStatusCompleted,
		Intent:     fhirModel.TaskIntentOrder,
		AuthoredOn: (*fhirModel.DateTime)(converto.TimePointer(time.Now().UTC())),
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

func prepareUpdatePatientBundle(
	params *fhirModel.Parameters, task *fhirModel.Task, patient *fhirModel.Patient) *fhirModel.Bundle {
	//nolint:dupl
	b := &fhirModel.Bundle{
		Resource: fhirModel.Resource{
			ID:           fhirModel.ID(uuid.NewV4().String()),
			ResourceType: fhirModel.ResourceBundle,
		},
		Type: fhirModel.BundleTypeTransaction,
		Entry: []*fhirModel.BundleEntry{
			{
				Resource: params,
				Request: &fhirModel.BundleEntryRequest{
					Method: http.MethodPost,
					URL:    fhirModel.ResourceParameters,
				},
			},
			{
				Resource: task,
				Request: &fhirModel.BundleEntryRequest{
					Method: http.MethodPost,
					URL:    fhirModel.ResourceTask,
				},
			},
			{
				Resource: patient,
				Request: &fhirModel.BundleEntryRequest{
					Method: http.MethodPut,
					URL:    fmt.Sprintf("%s/%s", fhirModel.ResourcePatient, patient.ID),
				},
			},
		},
	}

	return b
}

func (uc *UseCase) validateParametersCount(ctx context.Context, p *fhirModel.Patient) error {
	if p.MaritalStatus != nil ||
		len(p.Contact) != 0 ||
		len(p.Communication) != 0 ||
		len(p.Extension) != 0 {
		return nil
	}

	return cerror.NewValidationError(ctx, map[string]string{
		"Parameters.parameter.resource[0]": fmt.Sprintf("at least one of the parameters should be present: %v",
			strings.Join(entity.UpdatePatientParametersList(), ","))}).LogError()
}

func setPatientParams(dbPatient, patient *fhirModel.Patient) {
	if patient.MaritalStatus != nil {
		dbPatient.MaritalStatus = patient.MaritalStatus
	}

	if len(patient.Communication) != 0 {
		dbPatient.Communication = patient.Communication
	}

	if len(patient.Contact) != 0 {
		dbPatient.Contact = patient.Contact
	}

	for i := range patient.Extension {
		param := patient.Extension[i]
		if checkIsURLInList(param.URL) {
			isExist := false

			for j := range dbPatient.Extension {
				if dbPatient.Extension[j].URL == param.URL {
					dbPatient.Extension[j] = param
					isExist = true

					break
				}
			}

			if !isExist {
				dbPatient.Extension = append(dbPatient.Extension, param)
			}
		}
	}
}

func checkIsURLInList(urlReq string) bool {
	urlList := entity.UpdatePatientURLList()
	for i := range urlList {
		if urlList[i] == urlReq {
			return true
		}
	}

	return false
}
