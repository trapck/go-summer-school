package usecase

import (
	"context"
	"fmt"
	"strings"

	"wasfaty.api/pkg/cerror"
	fhirModel "wasfaty.api/pkg/fhir/model"
	"wasfaty.api/services/mpi/entity"
)

func (uc *UseCase) UpdatePatientEmail(ctx context.Context, id fhirModel.ID, p *fhirModel.Parameters) (*fhirModel.Task, error) {
	patient, err := uc.unmarshalPatientParam(ctx, p, 0)
	if err != nil {
		return nil, err
	}

	if patient.ID != id {
		return nil, cerror.NewF(ctx, cerror.KindBadValidation, "input id and patient id are not equal").LogError()
	}

	err = uc.validateParameterTelecomCount(ctx, patient)
	if err != nil {
		return nil, err
	}

	if pErr := uc.validatePatientProfile(ctx, p.Meta, patient.Meta); pErr != nil {
		return nil, pErr
	}

	if _, err = uc.fhir.ValidateParameters(ctx, p); err != nil {
		return nil, err
	}

	dbPatient, err := uc.fhir.GetPatientByID(ctx, id)
	if err != nil {
		return nil, err
	}

	err = uc.validatePatientByInternalRules(ctx, dbPatient)
	if err != nil {
		return nil, err
	}

	setPatientTelecomParams(dbPatient, patient)

	task := prepareUpdatePatientTask(p)

	b := prepareUpdatePatientBundle(p, task, dbPatient)

	_, err = uc.fhir.CreateBundle(ctx, b)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (uc *UseCase) validateParameterTelecomCount(ctx context.Context, p *fhirModel.Patient) error {
	if len(p.Telecom) == 1 {
		return nil
	}

	return cerror.NewValidationError(ctx, map[string]string{
		"Parameters.parameter.resource[0]": fmt.Sprintf("at least one of the parameters should be present: %v",
			strings.Join(entity.UpdateEmailPatientParametersList(), ","))}).LogError()
}

func setPatientTelecomParams(dbPatient, patient *fhirModel.Patient) {
	for i := range patient.Telecom {
		param := patient.Telecom[i]
		if param.System == "email" {
			isExist := false

			for j := range dbPatient.Telecom {
				if dbPatient.Telecom[j].System == param.System {
					dbPatient.Telecom[j].Value = param.Value
					isExist = true

					break
				}
			}

			if !isExist {
				dbPatient.Telecom = append(dbPatient.Telecom, param)
			}
		}
	}
}
