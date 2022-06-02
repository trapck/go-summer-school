package usecase

import (
	"context"

	fhirModel "wasfaty.api/pkg/fhir/model"
	"wasfaty.api/services/mpi/entity"
)

type FHIRClient interface {
	ValidateParameters(ctx context.Context, p *fhirModel.Parameters) (*fhirModel.OperationOutcome, error)
	CreateBundle(ctx context.Context, b *fhirModel.Bundle) (*fhirModel.Bundle, error)
	SearchTaskByParams(ctx context.Context, params *entity.SearchTaskParams) ([]*fhirModel.Task, error)
	GetTaskByID(ctx context.Context, id fhirModel.ID) (*fhirModel.Task, error)
	GetParametersByID(ctx context.Context, id fhirModel.ID) (*fhirModel.Parameters, error)
	SearchPatientByParams(ctx context.Context, params *entity.SearchPatientParams) ([]*fhirModel.Patient, error)
	GetPatientByID(ctx context.Context, id fhirModel.ID) (*fhirModel.Patient, error)
}

type OTPClient interface {
	GenerateByPhone(ctx context.Context, phone, processID string) (*entity.OTP, error)
	Validate(ctx context.Context, p *entity.ValidateOTPParams) error
}

type ExtDocRegistryClient interface {
	Search(ctx context.Context, p *fhirModel.Identifier) (*entity.ExtDocRegistrySearchResult, error)
}

type UseCase struct {
	fhir FHIRClient

	otp OTPClient

	docReg ExtDocRegistryClient
}

func New(fc FHIRClient, oc OTPClient, edrc ExtDocRegistryClient) *UseCase {
	return &UseCase{fhir: fc, otp: oc, docReg: edrc}
}
