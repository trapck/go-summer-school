package usecase

import (
	"context"
	"fmt"
	"net/http"

	uuid "github.com/satori/go.uuid"

	"wasfaty.api/pkg/cerror"
	"wasfaty.api/pkg/converto"
	fhirModel "wasfaty.api/pkg/fhir/model"
	"wasfaty.api/services/mpi/entity"
)

func (uc *UseCase) ConfirmCreatePatient(ctx context.Context, p *fhirModel.Parameters) (*fhirModel.Task, error) {
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

	patientParams, err := uc.getTaskParameters(ctx, t, 0)
	if err != nil {
		return nil, err
	}

	patient, err := uc.unmarshalPatientParam(ctx, patientParams, 0)
	if err != nil {
		return nil, err
	}

	telecom, err := uc.extractTelecomFromCreatePatientParams(ctx, patient)
	if err != nil {
		return nil, err
	}

	err = uc.validateOTP(ctx, op.TaskID, op.OTPCode, telecom.Value)
	if err != nil {
		return nil, err
	}

	err = uc.validatePatientDupls(ctx, patient)
	if err != nil {
		if cerror.ErrKind(err) == cerror.KindBadValidation {
			_ = uc.rejectTask(ctx, t, p)
		}

		return nil, err
	}

	err = uc.validatePatientFrauds(ctx, patient, telecom)
	if err != nil {
		return nil, err
	}

	uc.updateConfirmCreatePatientTask(t, patient, p)

	_, err = uc.saveConfirmCreatePatientBundle(ctx, t, patient, p)
	if err != nil {
		return nil, err
	}

	return t, nil
}

//nolint:dupl
func (uc *UseCase) updateConfirmCreatePatientTask(
	t *fhirModel.Task,
	patient *fhirModel.Patient,
	otpParams *fhirModel.Parameters) {
	t.Status = fhirModel.TaskStatusCompleted
	t.BusinessStatus = &fhirModel.CodeableConcept{Text: entity.TaskBusinessStatusPatientCreated}
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

func (uc *UseCase) saveConfirmCreatePatientBundle(
	ctx context.Context,
	t *fhirModel.Task,
	patient *fhirModel.Patient,
	otpParams *fhirModel.Parameters) (*fhirModel.Bundle, error) {
	//nolint:dupl
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
				Resource: patient,
				Request: &fhirModel.BundleEntryRequest{
					Method: http.MethodPost,
					URL:    fhirModel.ResourcePatient,
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

func (uc *UseCase) rejectTask(ctx context.Context, t *fhirModel.Task, p *fhirModel.Parameters) error {
	t.Status = fhirModel.TaskStatusRejected
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
				Reference: fmt.Sprintf("%s/%s", fhirModel.ResourceParameters, p.ID),
			},
		},
	})

	b := &fhirModel.Bundle{
		Resource: fhirModel.Resource{
			ID:           fhirModel.ID(uuid.NewV4().String()),
			ResourceType: fhirModel.ResourceBundle,
		},
		Type: fhirModel.BundleTypeTransaction,
		Entry: []*fhirModel.BundleEntry{
			{
				Resource: p,
				Request: &fhirModel.BundleEntryRequest{
					Method: http.MethodPost,
					URL:    fhirModel.ResourceParameters,
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

	_, err := uc.fhir.CreateBundle(ctx, b)

	return err
}

func (uc *UseCase) validatePatientDupls(ctx context.Context, p *fhirModel.Patient) error {
	patients, err := uc.searchPatientsByIdent(ctx, p.Identifier...)
	if err != nil {
		return err
	}

	if len(patients) > 0 {
		return cerror.NewF(ctx, cerror.KindBadValidation, "such person already exists").LogError()
	}

	return nil
}

func (uc *UseCase) validatePatientFrauds(ctx context.Context, p *fhirModel.Patient, t *fhirModel.ContactPoint) error {
	patients, err := uc.searchPatientsByPhone(ctx, p, t)
	if err != nil {
		return err
	}

	if len(patients) >= entity.MaxPatientsWithSamePhone {
		return cerror.NewF(ctx, cerror.KindBadValidation, "too many persons with same phone").LogError()
	}

	return nil
}

func (uc *UseCase) searchPatientsByPhone(ctx context.Context, p *fhirModel.Patient, t *fhirModel.ContactPoint) (
	[]*fhirModel.Patient, error) {
	if p.BirthDate == nil {
		return nil, cerror.NewF(ctx, cerror.KindBadValidation, "expected patient to have a birthdate")
	}

	patients, err := uc.fhir.SearchPatientByParams(ctx, &entity.SearchPatientParams{
		Phone: &entity.SearchPatientByPhoneParams{
			Phone:     t.Value,
			BirthDate: p.BirthDate.String(),
		},
	})

	if err != nil {
		return nil, err
	}

	return patients, nil
}

func (uc *UseCase) searchPatientsByIdent(ctx context.Context, identifiers ...*fhirModel.Identifier) ([]*fhirModel.Patient, error) {
	uc.sortIdentifiers(identifiers)

	for i, ident := range identifiers {
		if len(ident.Type.Codings) == 0 {
			return nil, cerror.NewF(ctx, cerror.KindBadValidation, "empty coding for identifier %d", i).LogError()
		}

		patients, err := uc.fhir.SearchPatientByParams(ctx, &entity.SearchPatientParams{
			Identifier: &entity.SearchPatientByIdentifierParams{
				Value: ident.Value,
				Type:  fmt.Sprintf("%s|%s", ident.Type.Codings[0].System, ident.Type.Codings[0].Code),
			},
		})

		if err != nil {
			return nil, err
		}

		if len(patients) > 0 {
			return patients, nil
		}
	}

	return []*fhirModel.Patient{}, nil
}

func (uc *UseCase) validateOTP(ctx context.Context, taskID fhirModel.ID, otp, phone string) error {
	err := uc.otp.Validate(ctx, &entity.ValidateOTPParams{
		Code:      otp,
		Value:     phone,
		ProcessID: string(taskID),
	})

	if err != nil && cerror.ErrKind(err) == cerror.KindNotExist {
		return cerror.NewF(ctx, cerror.KindNotExist, "otp for such patient request does not exist")
	}

	return err
}

func (uc *UseCase) extractTelecomFromCreatePatientParams(ctx context.Context, p *fhirModel.Patient) (
	*fhirModel.ContactPoint, error) {
	for _, telecom := range p.Telecom {
		if telecom.System == fhirModel.TelecomSystemPhone && telecom.Use == fhirModel.TelecomUseMobile {
			return telecom, nil
		}
	}

	return nil, cerror.NewF(ctx, cerror.KindBadValidation, "phone number for such patient request not found").LogError()
}

func (uc *UseCase) getTaskParameters(ctx context.Context, t *fhirModel.Task, paramsIndex int) (
	*fhirModel.Parameters, error) {
	if len(t.Input) < paramsIndex+1 {
		return nil, cerror.NewF(
			ctx,
			cerror.KindBadValidation,
			"expected task input to have at least %d elements",
			paramsIndex+1).
			LogError()
	}

	if t.Input[paramsIndex].ValueReference == nil {
		return nil, cerror.NewF(ctx, cerror.KindBadValidation, "task input valueReference is empty").LogError()
	}

	id, ok := t.Input[paramsIndex].ValueReference.ParseID()
	if !ok {
		return nil, cerror.NewF(
			ctx,
			cerror.KindBadValidation,
			"invalid task reference format: %s",
			t.Input[paramsIndex+1].ValueReference.Reference,
		).LogError()
	}

	params, err := uc.fhir.GetParametersByID(ctx, id)
	if err != nil {
		if cerror.ErrKind(err) == cerror.KindNotExist {
			return nil, cerror.NewF(ctx, cerror.KindNotExist, "parameters for such patient request do not exist")
		}

		return nil, err
	}

	return params, nil
}

func (uc *UseCase) validateConfirmRequestTask(ctx context.Context, t *fhirModel.Task) error {
	if t.Status != fhirModel.TaskStatusInProgress {
		return cerror.NewF(ctx, cerror.KindBadValidation, "such patient request is not active").LogError()
	}

	return nil
}

func (uc *UseCase) getTaskByID(ctx context.Context, id fhirModel.ID) (*fhirModel.Task, error) {
	t, err := uc.fhir.GetTaskByID(ctx, id)
	if err != nil && cerror.ErrKind(err) == cerror.KindNotExist {
		return nil, cerror.NewF(ctx, cerror.KindNotExist, "such patient request does not exist")
	}

	return t, err
}

func (uc *UseCase) extractConfirmRequestParams(ctx context.Context, p *fhirModel.Parameters) (
	*entity.ConfirmRequestParameters, error) {
	var otp string

	var taskRef *fhirModel.Reference

	for _, param := range p.Parameter {
		switch param.Name {
		case "otp":
			otp = converto.StringValue(param.ValueString)
		case "task_id":
			taskRef = param.ValueReference
		}
	}

	if otp == "" {
		return nil, cerror.NewValidationError(
			ctx, map[string]string{"Parameters.parameter": "missing otp parameter"}).LogError()
	}

	if taskRef == nil {
		return nil, cerror.NewValidationError(
			ctx, map[string]string{"Parameters.parameter": "missing task_id parameter"}).LogError()
	}

	taskID, ok := taskRef.ParseID()
	if !ok {
		return nil, cerror.NewValidationError(
			ctx, map[string]string{"Parameters.parameter": "invalid task_id parameter format"}).LogError()
	}

	return &entity.ConfirmRequestParameters{OTPCode: otp, TaskID: taskID}, nil
}
