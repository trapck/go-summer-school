package fhir_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"wasfaty.api/pkg/env"
	"wasfaty.api/pkg/fhir/client"
	fhirModel "wasfaty.api/pkg/fhir/model"
	"wasfaty.api/pkg/log"
	"wasfaty.api/services/mpi/adapter/api/fhir"
	"wasfaty.api/services/mpi/entity"
)

type fhirTestSuite struct {
	suite.Suite
	fhir       *fhir.Client
	mockClient *mockClient
}

type mockClient struct {
	validateResourceFunc       func(ctx context.Context, resName string, res fhirModel.AnyResource) (*fhirModel.OperationOutcome, error)
	createBundleFunc           func(ctx context.Context, b *fhirModel.Bundle) (*fhirModel.Bundle, error)
	getResourceByIDFunc        func(ctx context.Context, resName string, id fhirModel.ID, dst fhirModel.AnyResource) error
	searchResourceByParamsFunc func(ctx context.Context, resName string, params []*client.QParam) (*fhirModel.Bundle, error)
}

func (m *mockClient) ValidateResource(ctx context.Context, resName string, res fhirModel.AnyResource) (
	*fhirModel.OperationOutcome, error) {
	return m.validateResourceFunc(ctx, resName, res)
}

func (m *mockClient) CreateBundle(ctx context.Context, b *fhirModel.Bundle) (*fhirModel.Bundle, error) {
	return m.createBundleFunc(ctx, b)
}

func (m *mockClient) GetResourceByID(ctx context.Context, resName string, id fhirModel.ID, dst fhirModel.AnyResource) error {
	return m.getResourceByIDFunc(ctx, resName, id, dst)
}

func (m *mockClient) SearchResourceByParams(ctx context.Context, resName string, params []*client.QParam) (
	*fhirModel.Bundle, error) {
	return m.searchResourceByParamsFunc(ctx, resName, params)
}

func TestFHIRAdapterTestSuite(t *testing.T) {
	log.SetGlobalLogLevel("fatal")
	suite.Run(t, new(fhirTestSuite))
}

func (s *fhirTestSuite) SetupSuite() {
	s.mockClient = new(mockClient)
	s.fhir = fhir.NewClient(new(env.FHIR)).WithFHIR(s.mockClient)
}

func (s *fhirTestSuite) TearDownTest() {
	s.mockClient.validateResourceFunc = nil
	s.mockClient.createBundleFunc = nil
	s.mockClient.getResourceByIDFunc = nil
	s.mockClient.searchResourceByParamsFunc = nil
}

func (s *fhirTestSuite) TearDownSuite() {
}

func (s *fhirTestSuite) TestValidateParameters() {
	var isCalled bool

	o := &fhirModel.OperationOutcome{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{ID: fhirModel.ID("123")},
		}}

	s.mockClient.validateResourceFunc = func(_ context.Context, resName string, dst fhirModel.AnyResource) (
		*fhirModel.OperationOutcome, error) {
		isCalled = true

		s.IsType(&fhirModel.Parameters{}, dst)
		s.Equal(resName, "Parameters")

		return o, nil
	}

	r, err := s.fhir.ValidateParameters(context.Background(), new(fhirModel.Parameters))

	s.True(isCalled)
	s.NoError(err)
	s.Equal(o, r)
}

func (s *fhirTestSuite) TestCreateBundle() {
	var isCalled bool

	b := &fhirModel.Bundle{
		Resource: fhirModel.Resource{ID: fhirModel.ID("123")},
	}

	s.mockClient.createBundleFunc = func(_ context.Context, _ *fhirModel.Bundle) (*fhirModel.Bundle, error) {
		isCalled = true
		return b, nil
	}

	r, err := s.fhir.CreateBundle(context.Background(), b)

	s.True(isCalled)
	s.NoError(err)
	s.Equal(b, r)
}

//nolint:dupl
func (s *fhirTestSuite) TestGetPatientByID() {
	var isCalled bool

	p := &fhirModel.Patient{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{ID: fhirModel.ID("123")},
		}}

	s.mockClient.getResourceByIDFunc = func(
		_ context.Context, resName string, id fhirModel.ID, dst fhirModel.AnyResource) error {
		isCalled = true

		s.Equal("Patient", resName)
		s.Equal(p.ID, id)
		s.IsType(&fhirModel.Patient{}, dst)

		*dst.(*fhirModel.Patient) = *p

		return nil
	}

	r, err := s.fhir.GetPatientByID(context.Background(), p.ID)

	s.True(isCalled)
	s.NoError(err)
	s.Equal(p, r)
}

//nolint:dupl
func (s *fhirTestSuite) TestGetTaskByID() {
	var isCalled bool

	t := &fhirModel.Task{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{ID: fhirModel.ID("123")},
		}}

	s.mockClient.getResourceByIDFunc = func(
		_ context.Context, resName string, id fhirModel.ID, dst fhirModel.AnyResource) error {
		isCalled = true

		s.Equal("Task", resName)
		s.Equal(t.ID, id)
		s.IsType(&fhirModel.Task{}, dst)

		*dst.(*fhirModel.Task) = *t

		return nil
	}

	r, err := s.fhir.GetTaskByID(context.Background(), t.ID)

	s.True(isCalled)
	s.NoError(err)
	s.Equal(t, r)
}

func (s *fhirTestSuite) TestGetParametersByID() {
	var isCalled bool

	p := &fhirModel.Parameters{
		Resource: fhirModel.Resource{ID: fhirModel.ID("123")},
	}

	s.mockClient.getResourceByIDFunc = func(
		_ context.Context, resName string, id fhirModel.ID, dst fhirModel.AnyResource) error {
		isCalled = true

		s.Equal("Parameters", resName)
		s.Equal(p.ID, id)
		s.IsType(&fhirModel.Parameters{}, dst)

		*dst.(*fhirModel.Parameters) = *p

		return nil
	}

	r, err := s.fhir.GetParametersByID(context.Background(), p.ID)

	s.True(isCalled)
	s.NoError(err)
	s.Equal(p, r)
}

func (s *fhirTestSuite) TestSearchTaskByParams() {
	var isCalled bool

	searchParams := &entity.SearchTaskParams{
		Telecom: &entity.SearchTaskByTelecomParams{
			Value:     "value",
			Use:       "use",
			System:    "system",
			BirthDate: "2000-01-01",
		},
		Identifier: &entity.SearchTaskByIdentifierParams{
			Value: "value",
			Type:  "type",
		},
	}

	expectedTask := &fhirModel.Task{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{ID: fhirModel.ID("123")},
		}}

	s.mockClient.searchResourceByParamsFunc = func(_ context.Context, resName string, params []*client.QParam) (
		*fhirModel.Bundle, error) {
		isCalled = true

		s.Equal("Task", resName)
		s.Equal([]*client.QParam{
			{Key: "parameter-resource-telecom", Value: searchParams.Telecom.Value},
			{Key: "parameter-resource-telecom-use", Value: searchParams.Telecom.Use},
			{Key: "parameter-resource-telecom-system", Value: searchParams.Telecom.System},
			{Key: "parameter-resource-birthdate", Value: searchParams.Telecom.BirthDate},
			{Key: "parameter-resource-identifier", Value: searchParams.Identifier.Value},
			{Key: "parameter-resource-identifier-type", Value: searchParams.Identifier.Type},
			{Key: "_revinclude", Value: "Task:input-reference"},
		}, params)

		return &fhirModel.Bundle{
			Entry: []*fhirModel.BundleEntry{
				{},
				{
					FullURL:  "Task/123",
					Resource: expectedTask,
				},
			},
		}, nil
	}

	r, err := s.fhir.SearchTaskByParams(context.Background(), searchParams)

	s.True(isCalled)
	s.NoError(err)
	s.Equal(1, len(r))
	s.Equal(expectedTask, r[0])

	r, err = s.fhir.SearchTaskByParams(context.Background(), &entity.SearchTaskParams{})
	s.Error(err)
	s.Equal("no search criteria set", err.Error())
	s.Nil(r)
}

func (s *fhirTestSuite) TestSearchPatientByParams() {
	var isCalled bool

	searchParams := &entity.SearchPatientParams{
		Phone: &entity.SearchPatientByPhoneParams{
			Phone:     "phone",
			BirthDate: "2000-01-01",
		},
		Identifier: &entity.SearchPatientByIdentifierParams{
			Value: "value",
			Type:  "type",
		},
	}

	expectedPatient := &fhirModel.Patient{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{ID: fhirModel.ID("123")},
		}}

	s.mockClient.searchResourceByParamsFunc = func(_ context.Context, resName string, params []*client.QParam) (
		*fhirModel.Bundle, error) {
		isCalled = true

		s.Equal("Task", resName)
		s.Equal([]*client.QParam{
			{Key: "phone", Value: searchParams.Phone.Phone},
			{Key: "birthdate", Value: searchParams.Phone.BirthDate},
			{Key: "identifier", Value: searchParams.Identifier.Value},
			{Key: "identifier-type", Value: searchParams.Identifier.Type},
			{Key: "active", Value: "true"},
			{Key: "_profile", Value: fhirModel.StructureDefinitionPatientIdentified},
		}, params)

		return &fhirModel.Bundle{
			Entry: []*fhirModel.BundleEntry{
				{},
				{
					FullURL:  "Patient/123",
					Resource: expectedPatient,
				},
			},
		}, nil
	}

	r, err := s.fhir.SearchPatientByParams(context.Background(), searchParams)

	s.True(isCalled)
	s.NoError(err)
	s.Equal(1, len(r))
	s.Equal(expectedPatient, r[0])

	r, err = s.fhir.SearchPatientByParams(context.Background(), &entity.SearchPatientParams{})
	s.Error(err)
	s.Equal("no search criteria set", err.Error())
	s.Nil(r)
}
