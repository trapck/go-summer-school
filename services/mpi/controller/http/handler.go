package http

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"

	"wasfaty.api/pkg/cerror"
	"wasfaty.api/pkg/fhir/ferror"
	fhirModel "wasfaty.api/pkg/fhir/model"
)

type UseCase interface {
	CreatePatient(ctx context.Context, p *fhirModel.Parameters) (*fhirModel.Task, error)
	ConfirmCreatePatient(ctx context.Context, p *fhirModel.Parameters) (*fhirModel.Task, error)
	UpdatePatient(ctx context.Context, id fhirModel.ID, p *fhirModel.Parameters) (*fhirModel.Task, error)
	UpdatePatientIdentity(ctx context.Context, id fhirModel.ID, p *fhirModel.Parameters) (*fhirModel.Task, error)
	ConfirmUpdatePatientIdentity(ctx context.Context, id fhirModel.ID, p *fhirModel.Parameters) (*fhirModel.Task, error)
	UpdatePatientEmail(ctx context.Context, id fhirModel.ID, p *fhirModel.Parameters) (*fhirModel.Task, error)
}

var (
	errEmptyID = fmt.Errorf("empty id parameter")
)

type handler struct {
	uc UseCase
}

// newHandler creates new handler instance
func newHandler(uc UseCase) *handler {
	return &handler{uc: uc}
}

// (POST /Patient/$create-request)
//nolint:dupl
func (h *handler) createPatient(ctx *fiber.Ctx) error {
	req := new(fhirModel.Parameters)
	if err := ctx.BodyParser(req); err != nil {
		return writeErrorResp(ctx, cerror.New(ctx.Context(), cerror.KindBadParams, err).LogError())
	}

	if err := validateReqProfile(ctx.Context(), req, []string{fhirModel.StructureDefinitionPatientCreateRequest}); err != nil {
		return writeErrorResp(ctx, err)
	}

	resp, err := h.uc.CreatePatient(ctx.Context(), req)
	if err != nil {
		return writeErrorResp(ctx, err)
	}

	return ctx.Status(fiber.StatusOK).JSON(resp)
}

//nolint:dupl
// (POST /Patient/[id]/$update)
func (h *handler) updatePatient(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return writeErrorResp(ctx, cerror.New(ctx.Context(), cerror.KindBadParams, errEmptyID).LogError())
	}

	req := new(fhirModel.Parameters)
	if err := ctx.BodyParser(req); err != nil {
		return writeErrorResp(ctx, cerror.New(ctx.Context(), cerror.KindBadParams, err).LogError())
	}

	if err := validateReqProfile(ctx.Context(), req, []string{fhirModel.StructureDefinitionPatientUpdateRequest}); err != nil {
		return writeErrorResp(ctx, err)
	}

	resp, err := h.uc.UpdatePatient(ctx.Context(), fhirModel.ID(id), req)
	if err != nil {
		return writeErrorResp(ctx, err)
	}

	return ctx.Status(fiber.StatusOK).JSON(resp)
}

//nolint:dupl
// (POST /Patient/:id/$update-email)
func (h *handler) updatePatientEmail(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return writeErrorResp(ctx, cerror.New(ctx.Context(), cerror.KindBadParams, errEmptyID).LogError())
	}

	req := new(fhirModel.Parameters)
	if err := ctx.BodyParser(req); err != nil {
		return writeErrorResp(ctx, cerror.New(ctx.Context(), cerror.KindBadParams, err).LogError())
	}

	if err := validateReqProfile(ctx.Context(), req, []string{fhirModel.StructureDefinitionPatientUpdateEmailRequest}); err != nil {
		return writeErrorResp(ctx, err)
	}

	resp, err := h.uc.UpdatePatientEmail(ctx.Context(), fhirModel.ID(id), req)
	if err != nil {
		return writeErrorResp(ctx, err)
	}

	return ctx.Status(fiber.StatusOK).JSON(resp)
}

//nolint:dupl
// (POST /Patient/[id]/$update-identity)
func (h *handler) updatePatientIdentity(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return writeErrorResp(ctx, cerror.New(ctx.Context(), cerror.KindBadParams, errEmptyID).LogError())
	}

	req := new(fhirModel.Parameters)
	if err := ctx.BodyParser(req); err != nil {
		return writeErrorResp(ctx, cerror.New(ctx.Context(), cerror.KindBadParams, err).LogError())
	}

	if err := validateReqProfile(ctx.Context(), req, []string{fhirModel.StructureDefinitionPatientUpdateIdentityRequest}); err != nil {
		return writeErrorResp(ctx, err)
	}

	resp, err := h.uc.UpdatePatientIdentity(ctx.Context(), fhirModel.ID(id), req)
	if err != nil {
		return writeErrorResp(ctx, err)
	}

	return ctx.Status(fiber.StatusOK).JSON(resp)
}

// (POST /Patient/$confirm-create-request)
//nolint:dupl
func (h *handler) confirmCreatePatient(ctx *fiber.Ctx) error {
	req := new(fhirModel.Parameters)
	if err := ctx.BodyParser(req); err != nil {
		return writeErrorResp(ctx, cerror.New(ctx.Context(), cerror.KindBadParams, err).LogError())
	}

	if err := validateReqProfile(ctx.Context(), req, []string{fhirModel.StructureDefinitionPatientConfirmCreateRequest}); err != nil {
		return writeErrorResp(ctx, err)
	}

	resp, err := h.uc.ConfirmCreatePatient(ctx.Context(), req)
	if err != nil {
		return writeErrorResp(ctx, err)
	}

	return ctx.Status(fiber.StatusCreated).JSON(resp)
}

// (POST /Patient/[id]/$confirm-identity)
//nolint:dupl
func (h *handler) confirmUpdatePatientIdentity(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return writeErrorResp(ctx, cerror.New(ctx.Context(), cerror.KindBadParams, errEmptyID).LogError())
	}

	req := new(fhirModel.Parameters)
	if err := ctx.BodyParser(req); err != nil {
		return writeErrorResp(ctx, cerror.New(ctx.Context(), cerror.KindBadParams, err).LogError())
	}

	if err := validateReqProfile(ctx.Context(), req, []string{fhirModel.StructureDefinitionPatientConfirmUpdateIdentityRequest}); err != nil {
		return writeErrorResp(ctx, err)
	}

	resp, err := h.uc.ConfirmUpdatePatientIdentity(ctx.Context(), fhirModel.ID(id), req)
	if err != nil {
		return writeErrorResp(ctx, err)
	}

	return ctx.Status(fiber.StatusOK).JSON(resp)
}

func validateReqProfile(ctx context.Context, p *fhirModel.Parameters, allowedProfiles []string) error {
	if p.Meta == nil {
		return cerror.NewValidationError(ctx, map[string]string{"Parameters.meta": "value is required"})
	}

	if len(p.Meta.Profile) != 1 {
		return cerror.NewValidationError(ctx, map[string]string{"Parameters.meta.profile": "expected to have 1 value"})
	}

	actualP := p.Meta.Profile[0]
	for _, p := range allowedProfiles {
		if p == actualP {
			return nil
		}
	}

	return cerror.NewValidationError(ctx, map[string]string{"Parameters.meta.profile[0]": "given profile is not supported"})
}

func writeErrorResp(ctx *fiber.Ctx, err error) error {
	return ctx.Status(cerror.ErrKind(err).HTTPCode()).JSON(ferror.OutcomeFromError(err))
}
