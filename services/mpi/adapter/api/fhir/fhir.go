package fhir

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/valyala/fasthttp"

	"wasfaty.api/pkg/cerror"
	"wasfaty.api/pkg/env"
	"wasfaty.api/pkg/fhir/client"
	fhirModel "wasfaty.api/pkg/fhir/model"
	"wasfaty.api/services/mpi/entity"
)

type Client struct {
	cfg  *env.FHIR
	fhir FHIR
}

type FHIR interface {
	ValidateResource(ctx context.Context, resName string, res fhirModel.AnyResource) (*fhirModel.OperationOutcome, error)

	CreateBundle(ctx context.Context, b *fhirModel.Bundle) (*fhirModel.Bundle, error)
	GetResourceByID(ctx context.Context, resName string, id fhirModel.ID, dst fhirModel.AnyResource) error
	SearchResourceByParams(ctx context.Context, resName string, params []*client.QParam) (*fhirModel.Bundle, error)
}

func NewClient(cfg *env.FHIR) *Client {
	return &Client{cfg: cfg, fhir: client.New(cfg).WithHTTPClient(&fasthttp.Client{})}
}

func (c *Client) WithFHIR(f FHIR) *Client {
	c.fhir = f
	return c
}

func (c *Client) ValidateParameters(ctx context.Context, p *fhirModel.Parameters) (*fhirModel.OperationOutcome, error) {
	return c.fhir.ValidateResource(ctx, fhirModel.ResourceParameters, p)
}

func (c *Client) CreateBundle(ctx context.Context, b *fhirModel.Bundle) (*fhirModel.Bundle, error) {
	return c.fhir.CreateBundle(ctx, b)
}

func (c *Client) SearchTaskByParams(ctx context.Context, params *entity.SearchTaskParams) ([]*fhirModel.Task, error) {
	qParams := []*client.QParam{}
	if params.Telecom != nil {
		qParams = append(qParams, []*client.QParam{
			{Key: "parameter-resource-telecom", Value: params.Telecom.Value},
			{Key: "parameter-resource-telecom-use", Value: params.Telecom.Use},
			{Key: "parameter-resource-telecom-system", Value: params.Telecom.System},
			{Key: "parameter-resource-birthdate", Value: params.Telecom.BirthDate},
		}...)
	}

	if params.Identifier != nil {
		qParams = append(qParams, []*client.QParam{
			{Key: "parameter-resource-identifier", Value: params.Identifier.Value},
			{Key: "parameter-resource-identifier-type", Value: params.Identifier.Type},
		}...)
	}

	if len(qParams) == 0 {
		return nil, cerror.NewF(ctx, cerror.KindInternal, "no search criteria set").LogError()
	}

	qParams = append(qParams, &client.QParam{Key: "_revinclude", Value: "Task:input-reference"})

	bundle, err := c.fhir.SearchResourceByParams(ctx, fhirModel.ResourceTask, qParams)
	if err != nil {
		return nil, err
	}

	tasks := []*fhirModel.Task{}

	for i, e := range bundle.Entry {
		if strings.Contains(e.FullURL, fhirModel.ResourceTask) {
			task := new(fhirModel.Task)
			if err := interfaceToStruct(ctx, e.Resource, task, fmt.Sprintf("Entries.entry[%d].resource", i)); err != nil {
				return nil, err
			}
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

func (c *Client) GetPatientByID(ctx context.Context, id fhirModel.ID) (*fhirModel.Patient, error) {
	dst := new(fhirModel.Patient)
	if err := c.fhir.GetResourceByID(ctx, fhirModel.ResourcePatient, id, dst); err != nil {
		return nil, err
	}

	return dst, nil
}

func (c *Client) GetTaskByID(ctx context.Context, id fhirModel.ID) (*fhirModel.Task, error) {
	dst := new(fhirModel.Task)
	if err := c.fhir.GetResourceByID(ctx, fhirModel.ResourceTask, id, dst); err != nil {
		return nil, err
	}

	return dst, nil
}

func (c *Client) GetParametersByID(ctx context.Context, id fhirModel.ID) (*fhirModel.Parameters, error) {
	dst := new(fhirModel.Parameters)
	if err := c.fhir.GetResourceByID(ctx, fhirModel.ResourceParameters, id, dst); err != nil {
		return nil, err
	}

	return dst, nil
}

func (c *Client) SearchPatientByParams(ctx context.Context, params *entity.SearchPatientParams) ([]*fhirModel.Patient, error) {
	qParams := []*client.QParam{}
	if params.Phone != nil {
		qParams = append(qParams, []*client.QParam{
			{Key: "phone", Value: params.Phone.Phone},
			{Key: "birthdate", Value: params.Phone.BirthDate},
		}...)
	}

	if params.Identifier != nil {
		qParams = append(qParams, []*client.QParam{
			{Key: "identifier", Value: params.Identifier.Value},
			{Key: "identifier-type", Value: params.Identifier.Type},
		}...)
	}

	if len(qParams) == 0 {
		return nil, cerror.NewF(ctx, cerror.KindInternal, "no search criteria set").LogError()
	}

	qParams = append(qParams, []*client.QParam{
		{Key: "active", Value: "true"},
		{Key: "_profile", Value: fhirModel.StructureDefinitionPatientIdentified},
	}...)

	bundle, err := c.fhir.SearchResourceByParams(ctx, fhirModel.ResourceTask, qParams)
	if err != nil {
		return nil, err
	}

	patients := []*fhirModel.Patient{}

	for i, e := range bundle.Entry {
		if strings.Contains(e.FullURL, fhirModel.ResourcePatient) {
			patient := new(fhirModel.Patient)
			if err := interfaceToStruct(ctx, e.Resource, patient, fmt.Sprintf("Entries.entry[%d].resource", i)); err != nil {
				return nil, err
			}
			patients = append(patients, patient)
		}
	}

	return patients, nil
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
