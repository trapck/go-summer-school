package usecase

import (
	"context"
	"fmt"
	"net/http"

	uuid "github.com/satori/go.uuid"
	"wasfaty.api/pkg/cerror"
	fhirModel "wasfaty.api/pkg/fhir/model"
	"wasfaty.api/services/mpi/entity"
)

func (uc *UseCase) ConfirmUpdatePatientIdentity(ctx context.Context, id fhirModel.ID, p *fhirModel.Parameters) (
	*fhirModel.Task, error) {
	if _, err := uc.fhir.ValidateParameters(ctx, p); err != nil {
		return nil, err
	}

	op, err := uc.extractConfirmRequestParams(ctx, p)
	if err != nil {
		return nil, err
	}

	t, err := uc.getTaskByID(ctx, op.TaskID)
	if err != nil {
		return nil, err
	}

	err = uc.validateConfirmRequestTask(ctx, t)
	if err != nil {
		return nil, err
	}

	resourceParams, err := uc.getTaskParameters(ctx, t, 0)
	if err != nil {
		return nil, err
	}

	patientParams, err := uc.extractUpdatePatientIdentityParams(ctx, resourceParams)
	if err != nil {
		return nil, err
	}

	if patientParams.Patient.ID != id {
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

	err = uc.validateOTP(ctx, op.TaskID, op.OTPCode, patientParams.ConfirmationMethod)
	if err != nil {
		return nil, err
	}

	err = uc.validatePatientDupls(ctx, patientParams.Patient)
	if err != nil {
		if cerror.ErrKind(err) == cerror.KindBadValidation {
			_ = uc.rejectTask(ctx, t, p)
		}

		return nil, err
	}

	patient, err := uc.mergePatient(ctx, dbPatient, patientParams.Patient)
	if err != nil {
		return nil, err
	}

	uc.updateConfirmUpdatePatientIdentityTask(t, patient, p)

	_, err = uc.saveConfirmUpdatePatientIdentityBundle(ctx, t, patient, p)
	if err != nil {
		return nil, err
	}

	return t, nil
}

//nolint:dupl
func (uc *UseCase) updateConfirmUpdatePatientIdentityTask(
	t *fhirModel.Task,
	patient *fhirModel.Patient,
	otpParams *fhirModel.Parameters) {
	t.Status = fhirModel.TaskStatusCompleted
	t.BusinessStatus = &fhirModel.CodeableConcept{Text: entity.TaskBusinessStatusConfirmPatientIdentityUpdated}
	t.Input = append(t.Input, &fhirModel.TaskInput{
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
				Reference: fmt.Sprintf("%s/%s", fhirModel.ResourceParameters, otpParams.ID),
			},
		},
	})

	t.Output = append(t.Output, &fhirModel.TaskOutput{
		Type: &fhirModel.CodeableConcept{
			Codings: []*fhirModel.Coding{
				{
					Code:   fhirModel.ResourcePatient,
					System: fhirModel.CodingSystemResourceTypes,
				},
			},
		},
		ValueX: fhirModel.ValueX{
			ValueReference: &fhirModel.Reference{
				Reference: fmt.Sprintf("%s/%s", fhirModel.ResourcePatient, patient.ID),
			},
		},
	})
}

func (uc *UseCase) saveConfirmUpdatePatientIdentityBundle(
	ctx context.Context,
	t *fhirModel.Task,
	p *fhirModel.Patient,
	otpParams *fhirModel.Parameters) (*fhirModel.Bundle, error) {
	b := &fhirModel.Bundle{
		Resource: fhirModel.Resource{
			ID:           fhirModel.ID(uuid.NewV4().String()),
			ResourceType: fhirModel.ResourceBundle,
		},
		Type: fhirModel.BundleTypeTransaction,
		Entry: []*fhirModel.BundleEntry{
			{
				Resource: otpParams,
				Request: &fhirModel.BundleEntryRequest{
					Method: http.MethodPost,
					URL:    fhirModel.ResourceParameters,
				},
			},
			{
				Resource: p,
				Request: &fhirModel.BundleEntryRequest{
					Method: http.MethodPut,
					URL:    fmt.Sprintf("%s/%s", fhirModel.ResourcePatient, p.ID),
				},
			},
			{
				Resource: t,
				Request: &fhirModel.BundleEntryRequest{
					Method: http.MethodPut,
					URL:    fmt.Sprintf("%s/%s", fhirModel.ResourceTask, t.ID),
				},
			},
		},
	}

	return uc.fhir.CreateBundle(ctx, b)
}

func (uc *UseCase) mergePatient(ctx context.Context, dbPatient, newParams *fhirModel.Patient) (*fhirModel.Patient, error) {
	excludeParams := []string{"meta", "extension"}
	dbPatient.Extension = uc.mergeExtensions(dbPatient.Extension, newParams.Extension)
	dbPatientParams := mapFromStruct(dbPatient)
	newPatientParams := mapFromStruct(newParams)

	for i := range newPatientParams {
		if !contains(excludeParams, i) {
			dbPatientParams[i] = newPatientParams[i]
		}
	}

	patient := new(fhirModel.Patient)
	err := mapToStruct(ctx, dbPatientParams, patient, "")

	if err != nil {
		return nil, err
	}

	return patient, nil
}

func (uc *UseCase) mergeExtensions(extensions, newExtensions []*fhirModel.Extension) []*fhirModel.Extension {
	var result []*fhirModel.Extension
	result = append(result, extensions...)

	for i := range newExtensions {
		isExist := false

		for j := range result {
			if result[j].URL == newExtensions[i].URL {
				result = append(result, newExtensions[i])
				isExist = true

				break
			}
		}

		if !isExist {
			result = append(result, newExtensions[i])
		}
	}

	return result
}
