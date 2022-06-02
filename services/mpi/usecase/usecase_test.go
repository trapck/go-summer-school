package usecase_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	uuid "github.com/satori/go.uuid"
	"wasfaty.api/pkg/cerror"
	"wasfaty.api/pkg/converto"
	fhirModel "wasfaty.api/pkg/fhir/model"
	"wasfaty.api/pkg/log"
	"wasfaty.api/services/mpi/entity"
	"wasfaty.api/services/mpi/usecase"
)

type createPatientTestFHIR struct {
	bundles      []*fhirModel.Bundle
	tasks        []*fhirModel.Task
	patients     []*fhirModel.Patient
	duplPatients []*fhirModel.Patient
	parameters   []*fhirModel.Parameters

	validateParametersCallsCount int

	searchTaskByParamsArgs    []*entity.SearchTaskParams
	searchPatientByParamsArgs []*entity.SearchPatientParams
}

func (c *createPatientTestFHIR) ValidateParameters(ctx context.Context, p *fhirModel.Parameters) (*fhirModel.OperationOutcome, error) {
	c.validateParametersCallsCount++
	return new(fhirModel.OperationOutcome), nil
}

func (c *createPatientTestFHIR) CreateBundle(ctx context.Context, b *fhirModel.Bundle) (*fhirModel.Bundle, error) {
	c.bundles = append(c.bundles, b)

	return b, nil
}

func (c *createPatientTestFHIR) SearchTaskByParams(ctx context.Context, params *entity.SearchTaskParams) (
	[]*fhirModel.Task, error) {
	c.searchTaskByParamsArgs = append(c.searchTaskByParamsArgs, params)
	return c.tasks, nil
}

func (c *createPatientTestFHIR) GetPatientByID(ctx context.Context, id fhirModel.ID) (*fhirModel.Patient, error) {
	for _, p := range c.patients {
		if p.ID == id {
			return p, nil
		}
	}

	return nil, cerror.New(ctx, cerror.KindNotExist, cerror.ErrNotFound)
}

func (c *createPatientTestFHIR) GetTaskByID(ctx context.Context, id fhirModel.ID) (*fhirModel.Task, error) {
	for _, t := range c.tasks {
		if t.ID == id {
			return t, nil
		}
	}

	return nil, cerror.New(ctx, cerror.KindNotExist, cerror.ErrNotFound)
}

func (c *createPatientTestFHIR) GetParametersByID(ctx context.Context, id fhirModel.ID) (*fhirModel.Parameters, error) {
	for _, p := range c.parameters {
		if p.ID == id {
			return p, nil
		}
	}

	return nil, cerror.New(ctx, cerror.KindNotExist, cerror.ErrNotFound)
}

func (c *createPatientTestFHIR) SearchPatientByParams(ctx context.Context, params *entity.SearchPatientParams) (
	[]*fhirModel.Patient, error) {
	c.searchPatientByParamsArgs = append(c.searchPatientByParamsArgs, params)
	return c.duplPatients, nil
}

type createPatientTestExtDocReg struct {
}

func (c *createPatientTestExtDocReg) Search(ctx context.Context, i *fhirModel.Identifier) (
	*entity.ExtDocRegistrySearchResult, error) {
	return &entity.ExtDocRegistrySearchResult{IsValid: true}, nil
}

type createPatientTestOTP struct {
	otps map[string]*entity.OTP
}

func (c *createPatientTestOTP) GenerateByPhone(ctx context.Context, phone, processID string) (*entity.OTP, error) {
	o := &entity.OTP{
		Code:  "1234",
		Value: phone,
	}
	c.otps[processID] = o

	return o, nil
}

func (c *createPatientTestOTP) Validate(ctx context.Context, p *entity.ValidateOTPParams) error {
	if o, ok := c.otps[p.ProcessID]; ok && o.Code == p.Code && o.Value == p.Value {
		return nil
	}

	return cerror.New(ctx, cerror.KindNotExist, cerror.ErrNotFound)
}

type useCaseTestSuite struct {
	suite.Suite
	fhir   *createPatientTestFHIR
	otp    *createPatientTestOTP
	extReg *createPatientTestExtDocReg
	uc     *usecase.UseCase
}

func TestUseCaseTestSuite(t *testing.T) {
	log.SetGlobalLogLevel("fatal")
	suite.Run(t, new(useCaseTestSuite))
}

func (s *useCaseTestSuite) SetupSuite() {
	s.fhir = new(createPatientTestFHIR)
	s.otp = new(createPatientTestOTP)
	s.extReg = new(createPatientTestExtDocReg)

	s.otp.otps = make(map[string]*entity.OTP)

	s.uc = usecase.New(s.fhir, s.otp, s.extReg)
}

func (s *useCaseTestSuite) TearDownTest() {
	s.fhir.patients = nil
	s.fhir.duplPatients = nil
	s.fhir.tasks = nil
	s.fhir.parameters = nil
	s.fhir.bundles = nil
	s.fhir.validateParametersCallsCount = 0
	s.fhir.searchTaskByParamsArgs = nil
	s.fhir.searchPatientByParamsArgs = nil

	s.otp.otps = make(map[string]*entity.OTP)
}

func (s *useCaseTestSuite) TearDownSuite() {
}

func (s *useCaseTestSuite) TestValidatePatientProfile() {
	for _, p := range []struct {
		req                 string
		patientParamIndex   int
		usecaseMethodCaller func(*fhirModel.Parameters) error
	}{
		{createPatientReqBody, 0, func(p *fhirModel.Parameters) error {
			_, err := s.uc.CreatePatient(context.Background(), p)
			return err
		}},
		{updatePatientIdentityReqBody, 1, func(p *fhirModel.Parameters) error {
			_, err := s.uc.UpdatePatientIdentity(context.Background(), fhirModel.ID("244a8e88-c0b0-4d60-b5d7-14afbe79f5f5"), p)
			return err
		}},
		{updatePatientReqBody, 0, func(p *fhirModel.Parameters) error {
			_, err := s.uc.UpdatePatient(context.Background(), fhirModel.ID("9e293127-8ffc-462c-aea0-d5464794b526"), p)
			return err
		}},
	} {
		s.validatePatientProfile(p.req, p.patientParamIndex, p.usecaseMethodCaller)
	}
}

func (s *useCaseTestSuite) validatePatientProfile(
	req string, patientParamIndex int, usecaseMethodCaller func(*fhirModel.Parameters) error) {
	p := new(fhirModel.Parameters)
	err := json.Unmarshal([]byte(req), p)
	s.NoError(err)

	patient := new(fhirModel.Patient)
	interfaceToStruct(p.Parameter[patientParamIndex].Resource, patient)

	patient.Meta.Profile[0] += "123"
	p.Parameter[patientParamIndex].Resource = patient
	err = usecaseMethodCaller(p)
	s.Error(err)

	vErr, ok := err.(*cerror.ValidationError)
	s.True(ok)
	s.Equal(map[string]string{
		"Resource.meta.profile[0]": "given patient profile is not supported for current operation",
	}, vErr.Payload())
}

func (s *useCaseTestSuite) TestCreatePatientSuccess() {
	p := new(fhirModel.Parameters)
	err := json.Unmarshal([]byte(createPatientReqBody), p)
	s.NoError(err)

	t, err := s.uc.CreatePatient(context.Background(), p)

	s.NoError(err)
	s.NotNil(t)

	s.Equal(1, s.fhir.validateParametersCallsCount)
	s.Equal(1, len(s.fhir.searchPatientByParamsArgs))
	s.Equal(1, len(s.fhir.bundles))
	s.Equal(2, len(s.fhir.bundles[0].Entry))

	bundleParameters, ok := s.fhir.bundles[0].Entry[0].Resource.(*fhirModel.Parameters)
	s.True(ok)
	s.Equal(p, bundleParameters)

	bundleTask, ok := s.fhir.bundles[0].Entry[1].Resource.(*fhirModel.Task)
	s.True(ok)
	s.Equal(t, bundleTask)

	//nolint:dupl
	expectedTask := &fhirModel.Task{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{
				ID:           t.ID,
				ResourceType: fhirModel.ResourceTask,
				Meta:         &fhirModel.Meta{Profile: []string{fhirModel.StructureDefinitionTaskPatientCreate}},
			},
		},

		Status:         fhirModel.TaskStatusInProgress,
		BusinessStatus: &fhirModel.CodeableConcept{Text: entity.TaskBusinessStatusOTPCodeSent},
		Intent:         fhirModel.TaskIntentOrder,
		AuthoredOn:     t.AuthoredOn,
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

	s.Equal(expectedTask, t)
	s.False(t.AuthoredOn.Time().IsZero())
	s.NotEmpty(t.ID)
}

func (s *useCaseTestSuite) TestCreatePatientValidateParameters() {
	ctx := context.Background()

	p := new(fhirModel.Parameters)
	err := json.Unmarshal([]byte(createPatientReqBody), p)
	s.NoError(err)

	patient := new(fhirModel.Patient)
	interfaceToStruct(p.Parameter[0].Resource, patient)

	s.NoError(err)

	patient.Active = converto.BoolPointer(false)
	p.Parameter[0].Resource = patient

	_, err = s.uc.CreatePatient(ctx, p)
	s.Error(err)
	s.Contains(err.Error(), "should have true value")

	patient.Active = converto.BoolPointer(true)
	patientResourceMap := mapFromStruct(patient)
	patientResourceMap["deceasedBoolean"] = nil
	patientResourceMap["photo"] = nil
	p.Parameter[0].Resource = patientResourceMap

	_, err = s.uc.CreatePatient(ctx, p)
	s.Error(err)
	s.Contains(err.Error(), "forbidden parameters: deceasedBoolean,photo")
}

func (s *useCaseTestSuite) TestCreatePatientInternalValidate() {
	ctx := context.Background()

	p := new(fhirModel.Parameters)
	err := json.Unmarshal([]byte(createPatientReqBody), p)
	s.NoError(err)

	patient := new(fhirModel.Patient)
	interfaceToStruct(p.Parameter[0].Resource, patient)

	s.NoError(err)

	ext := patient.Extension
	patient.Extension = nil
	p.Parameter[0].Resource = patient

	_, err = s.uc.CreatePatient(ctx, p)
	s.Error(err)
	s.Contains(err.Error(), "nationality is not passed")

	patient.Extension = ext
	patient.Identifier[0].Period.End = (*fhirModel.DateTime)(converto.TimePointer(time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)))
	p.Parameter[0].Resource = patient

	_, err = s.uc.CreatePatient(ctx, p)
	s.Error(err)
	s.Contains(err.Error(), "identifier is expired")

	patient.Identifier[0].Period.End = (*fhirModel.DateTime)(converto.TimePointer(time.Now().UTC()))
	validNIValue := patient.Identifier[0].Value
	patient.Identifier[0].Value = "1234567890"
	p.Parameter[0].Resource = patient

	_, err = s.uc.CreatePatient(ctx, p)
	s.Error(err)
	s.Contains("Parameters.parameter[0].resource.identifier[0].value:invalid identifier value format", err.Error())

	patient.Identifier[0].Value = validNIValue
	patient.Identifier[0].Type.Codings[0].Code = "some-code"
	p.Parameter[0].Resource = patient

	_, err = s.uc.CreatePatient(ctx, p)
	s.Error(err)
	s.Contains(err.Error(), "identifier code should be one of NI,DP,CZ,JHN")

	patient.Identifier[0].Type.Codings[0].Code = "NI"
	p.Parameter[0].Resource = patient

	_, err = s.uc.CreatePatient(ctx, p)
	s.NoError(err)
}

func (s *useCaseTestSuite) TestCreatePatientSearchDuplPatients() {
	ctx := context.Background()

	p := new(fhirModel.Parameters)
	err := json.Unmarshal([]byte(createPatientReqBody), p)
	s.NoError(err)

	patient := new(fhirModel.Patient)
	interfaceToStruct(p.Parameter[0].Resource, patient)

	_, err = s.uc.CreatePatient(ctx, p)
	s.NoError(err)
	s.Equal(1, len(s.fhir.searchPatientByParamsArgs))
	s.Equal(&entity.SearchPatientByIdentifierParams{
		Value: patient.Identifier[0].Value,
		Type: fmt.Sprintf("%s|%s",
			patient.Identifier[0].Type.Codings[0].System,
			patient.Identifier[0].Type.Codings[0].Code),
	}, s.fhir.searchPatientByParamsArgs[0].Identifier)

	s.fhir.duplPatients = []*fhirModel.Patient{patient}

	_, err = s.uc.CreatePatient(ctx, p)
	s.Error(err)
	s.Contains(err.Error(), "such person already exists")
}

func (s *useCaseTestSuite) TestCreatePatientSearchDuplTasks() {
	ctx := context.Background()

	p := new(fhirModel.Parameters)
	err := json.Unmarshal([]byte(createPatientReqBody), p)
	s.NoError(err)

	patient := new(fhirModel.Patient)
	interfaceToStruct(p.Parameter[0].Resource, patient)

	_, err = s.uc.CreatePatient(ctx, p)
	s.NoError(err)
	s.Equal(1, len(s.fhir.bundles))
	s.Equal(2, len(s.fhir.searchTaskByParamsArgs))
	s.Equal(&entity.SearchTaskByIdentifierParams{
		Value: patient.Identifier[0].Value,
		Type: fmt.Sprintf("%s|%s",
			patient.Identifier[0].Type.Codings[0].System,
			patient.Identifier[0].Type.Codings[0].Code),
	}, s.fhir.searchTaskByParamsArgs[0].Identifier)
	s.Nil(s.fhir.searchTaskByParamsArgs[0].Telecom)
	s.Equal(&entity.SearchTaskByTelecomParams{
		Value:     patient.Telecom[0].Value,
		Use:       patient.Telecom[0].Use,
		System:    patient.Telecom[0].System,
		BirthDate: patient.BirthDate.String(),
	}, s.fhir.searchTaskByParamsArgs[1].Telecom)
	s.Nil(s.fhir.searchTaskByParamsArgs[1].Identifier)

	s.fhir.tasks = []*fhirModel.Task{new(fhirModel.Task)}
	_, err = s.uc.CreatePatient(ctx, p)
	s.NoError(err)
	s.Equal(2, len(s.fhir.bundles))
	s.Equal(4, len(s.fhir.searchTaskByParamsArgs))

	s.fhir.tasks[0].ID = fhirModel.ID(uuid.NewV4().String())
	s.fhir.tasks[0].Status = fhirModel.TaskStatusInProgress
	_, err = s.uc.CreatePatient(ctx, p)
	s.NoError(err)
	s.Equal(3, len(s.fhir.bundles))
	s.Equal(5, len(s.fhir.searchTaskByParamsArgs))
	s.NotNil(s.fhir.searchTaskByParamsArgs[4].Identifier)
	s.Nil(s.fhir.searchTaskByParamsArgs[4].Telecom)
	s.Equal(3, len(s.fhir.bundles[2].Entry))

	t, ok := s.fhir.bundles[2].Entry[2].Resource.(*fhirModel.Task)

	s.True(ok)
	s.Equal(s.fhir.tasks[0].ID, t.ID)
	s.Equal(fhirModel.TaskStatusCanceled, t.Status)
}

func (s *useCaseTestSuite) TestCreatePatientGenerateOTP() {
	ctx := context.Background()

	p := new(fhirModel.Parameters)
	err := json.Unmarshal([]byte(createPatientReqBody), p)
	s.NoError(err)

	patient := new(fhirModel.Patient)
	interfaceToStruct(p.Parameter[0].Resource, patient)

	s.NoError(err)

	patient.Telecom[0].System = "not-existing-system"
	patient.Telecom[0].Use = "not-existing-use"
	p.Parameter[0].Resource = patient

	_, err = s.uc.CreatePatient(ctx, p)
	s.Error(err)

	cErr, ok := err.(*cerror.CError)

	s.True(ok)
	s.Contains(fmt.Sprintf("%v", cErr.Payload()), "no mobile phone found")

	patient.Telecom[0].System = fhirModel.TelecomSystemPhone
	p.Parameter[0].Resource = patient

	_, err = s.uc.CreatePatient(ctx, p)
	s.Error(err)

	cErr, ok = err.(*cerror.CError)

	s.True(ok)
	s.Contains(fmt.Sprintf("%v", cErr.Payload()), "no mobile phone found")

	patient.Telecom[0].Use = fhirModel.TelecomUseMobile
	p.Parameter[0].Resource = patient

	_, err = s.uc.CreatePatient(ctx, p)
	s.NoError(err)

	s.Equal(1, len(s.otp.otps))
}

func (s *useCaseTestSuite) TestConfirmCreatePatientSuccess() {
	p := new(fhirModel.Parameters)
	err := json.Unmarshal([]byte(confirmCreatePatientReqBody), p)
	s.NoError(err)

	patientParams := new(fhirModel.Parameters)
	err = json.Unmarshal([]byte(createPatientReqBody), patientParams)
	s.NoError(err)

	s.fhir.parameters = []*fhirModel.Parameters{patientParams}

	s.fhir.tasks = []*fhirModel.Task{{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{
				ID: fhirModel.ID("9e293127-8ffc-462c-aea0-d5464794b526"),
			},
		},
		Status: fhirModel.TaskStatusInProgress,
		Input: []*fhirModel.TaskInput{
			{
				ValueX: fhirModel.ValueX{
					ValueReference: &fhirModel.Reference{
						Reference: "Parameters/" + patientParams.ID.String(),
					},
				},
			},
		},
	}}
	s.otp.otps = map[string]*entity.OTP{
		s.fhir.tasks[0].ID.String(): {
			Code:  "2655",
			Value: "+380673212121",
		},
	}

	actualTask, err := s.uc.ConfirmCreatePatient(context.Background(), p)

	s.NoError(err)
	s.NotNil(actualTask)

	s.Equal(1, s.fhir.validateParametersCallsCount)
	s.Equal(1, len(s.otp.otps))
	s.Equal(1, len(s.fhir.bundles))
	s.Equal(3, len(s.fhir.bundles[0].Entry))

	bundleParameters, ok := s.fhir.bundles[0].Entry[0].Resource.(*fhirModel.Parameters)
	s.True(ok)
	s.Equal(p, bundleParameters)

	s.Equal(http.MethodPut, s.fhir.bundles[0].Entry[2].Request.Method)
	s.Equal("Task/9e293127-8ffc-462c-aea0-d5464794b526", s.fhir.bundles[0].Entry[2].Request.URL)

	bundleTask, ok := s.fhir.bundles[0].Entry[2].Resource.(*fhirModel.Task)

	s.True(ok)
	s.Equal(fhirModel.TaskStatusCompleted, bundleTask.Status)
	s.Equal(&fhirModel.CodeableConcept{Text: entity.TaskBusinessStatusPatientCreated}, bundleTask.BusinessStatus)
	s.Equal(2, len(bundleTask.Input))
	s.Equal("Parameters/"+p.ID.String(), bundleTask.Input[1].ValueReference.Reference)
	s.Equal(1, len(bundleTask.Output))
	s.Equal("Patient/"+patientParams.ID.String(), bundleTask.Output[0].ValueReference.Reference)

	s.Equal(bundleTask, actualTask)

	bundlePatient, ok := s.fhir.bundles[0].Entry[1].Resource.(*fhirModel.Patient)
	s.True(ok)

	patient := new(fhirModel.Patient)

	interfaceToStruct(patientParams.Parameter[0].Resource, patient)
	s.Equal(patient, bundlePatient)
}

func (s *useCaseTestSuite) TestConfirmCreatePatientInvalidParams() {
	p := new(fhirModel.Parameters)
	err := json.Unmarshal([]byte(confirmCreatePatientReqBody), p)
	s.NoError(err)

	otpParam := p.Parameter[0]
	taskParam := p.Parameter[1]

	p.Parameter = []*fhirModel.ParametersParameter{}

	_, err = s.uc.ConfirmCreatePatient(context.Background(), p)

	s.Error(err)

	cErr, ok := err.(*cerror.CError)
	s.True(ok)
	s.Contains(fmt.Sprintf("%v", cErr.Payload()), "missing otp parameter")

	p.Parameter = []*fhirModel.ParametersParameter{otpParam}

	_, err = s.uc.ConfirmCreatePatient(context.Background(), p)

	cErr, ok = err.(*cerror.CError)
	s.True(ok)
	s.Contains(fmt.Sprintf("%v", cErr.Payload()), "missing task_id parameter")

	p.Parameter = []*fhirModel.ParametersParameter{otpParam, taskParam}

	_, err = s.uc.ConfirmCreatePatient(context.Background(), p)

	cErr, ok = err.(*cerror.CError)
	s.True(ok)
	s.NotContains(fmt.Sprintf("%v", cErr.Payload()), "missing otp parameter")
	s.NotContains(fmt.Sprintf("%v", cErr.Payload()), "missing correct task_id parameter")
}

func (s *useCaseTestSuite) TestConfirmCreatePatientValidationErrors() {
	p := new(fhirModel.Parameters)
	err := json.Unmarshal([]byte(confirmCreatePatientReqBody), p)
	s.NoError(err)

	_, err = s.uc.ConfirmCreatePatient(context.Background(), p)

	cErr, ok := err.(*cerror.CError)
	s.True(ok)
	s.Equal(cerror.KindNotExist.String(), cErr.Kind().String())
	s.Contains(err.Error(), "such patient request does not exist")

	s.fhir.tasks = []*fhirModel.Task{{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{
				ID: fhirModel.ID("9e293127-8ffc-462c-aea0-d5464794b526"),
			},
		},
	}}

	_, err = s.uc.ConfirmCreatePatient(context.Background(), p)

	cErr, ok = err.(*cerror.CError)
	s.True(ok)
	s.Equal(cerror.KindBadValidation.String(), cErr.Kind().String())
	s.Contains(err.Error(), "such patient request is not active")

	s.fhir.tasks[0].Status = fhirModel.TaskStatusInProgress

	_, err = s.uc.ConfirmCreatePatient(context.Background(), p)

	cErr, ok = err.(*cerror.CError)
	s.True(ok)
	s.Equal(cerror.KindBadValidation.String(), cErr.Kind().String())
	s.Contains(err.Error(), "expected task input to have at least 1 elements")

	patientParams := new(fhirModel.Parameters)
	err = json.Unmarshal([]byte(createPatientReqBody), patientParams)
	s.NoError(err)

	s.fhir.tasks[0].Input = []*fhirModel.TaskInput{
		{
			ValueX: fhirModel.ValueX{
				ValueReference: &fhirModel.Reference{
					Reference: "Parameters/" + patientParams.ID.String(),
				},
			},
		},
	}

	_, err = s.uc.ConfirmCreatePatient(context.Background(), p)

	cErr, ok = err.(*cerror.CError)
	s.True(ok)
	s.Equal(cerror.KindNotExist.String(), cErr.Kind().String())
	s.Contains(err.Error(), "parameters for such patient request do not exist")

	patient := new(fhirModel.Patient)
	interfaceToStruct(patientParams.Parameter[0].Resource, patient)

	patient.Telecom[0].System = "not-existing-system"
	patientParams.Parameter[0].Resource = patient
	s.fhir.parameters = []*fhirModel.Parameters{patientParams}

	_, err = s.uc.ConfirmCreatePatient(context.Background(), p)

	cErr, ok = err.(*cerror.CError)
	s.True(ok)
	s.Equal(cerror.KindBadValidation.String(), cErr.Kind().String())
	s.Contains(err.Error(), "phone number for such patient request not found")

	patient.Telecom[0].System = fhirModel.TelecomSystemPhone
	patientParams.Parameter[0].Resource = patient
	s.fhir.parameters = []*fhirModel.Parameters{patientParams}

	_, err = s.uc.ConfirmCreatePatient(context.Background(), p)

	cErr, ok = err.(*cerror.CError)
	s.True(ok)
	s.Equal(cerror.KindNotExist.String(), cErr.Kind().String())
	s.Contains(err.Error(), "otp for such patient request does not exist")

	s.otp.otps = map[string]*entity.OTP{
		"9e293127-8ffc-462c-aea0-d5464794b526": {
			Code:  "2655",
			Value: "+380673212121",
		},
	}

	s.fhir.duplPatients = []*fhirModel.Patient{patient}
	_, err = s.uc.ConfirmCreatePatient(context.Background(), p)

	s.Error(err)
	s.Contains(err.Error(), "such person already exists")
	s.Equal(1, len(s.fhir.searchPatientByParamsArgs))
	s.Equal(&entity.SearchPatientParams{
		Identifier: &entity.SearchPatientByIdentifierParams{
			Value: patient.Identifier[0].Value,
			Type: fmt.Sprintf(
				"%s|%s",
				patient.Identifier[0].Type.Codings[0].System,
				patient.Identifier[0].Type.Codings[0].Code),
		},
	}, s.fhir.searchPatientByParamsArgs[0])

	s.Equal(1, len(s.fhir.bundles))
	s.Equal(2, len(s.fhir.bundles[0].Entry))

	t, ok := s.fhir.bundles[0].Entry[1].Resource.(*fhirModel.Task)

	s.True(ok)
	s.Equal(fhirModel.TaskStatusRejected, t.Status)

	s.fhir.duplPatients = nil
	s.fhir.searchPatientByParamsArgs = nil

	s.fhir.tasks = []*fhirModel.Task{{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{
				ID: fhirModel.ID("9e293127-8ffc-462c-aea0-d5464794b526"),
			},
		},
		Status: fhirModel.TaskStatusInProgress,
		Input: []*fhirModel.TaskInput{
			{
				ValueX: fhirModel.ValueX{
					ValueReference: &fhirModel.Reference{
						Reference: "Parameters/" + patientParams.ID.String(),
					},
				},
			},
		},
	}}

	_, err = s.uc.ConfirmCreatePatient(context.Background(), p)
	s.NoError(err)

	s.Equal(2, len(s.fhir.searchPatientByParamsArgs))
	s.Equal(&entity.SearchPatientParams{
		Phone: &entity.SearchPatientByPhoneParams{
			Phone:     patient.Telecom[0].Value,
			BirthDate: patient.BirthDate.String(),
		},
	}, s.fhir.searchPatientByParamsArgs[1])
}

func (s *useCaseTestSuite) TestUpdatePatient() {
	ctx := context.Background()
	id := fhirModel.ID(uuid.NewV4().String())

	_, err := s.uc.UpdatePatient(ctx, id, &fhirModel.Parameters{})
	s.Error(err)

	cErr, ok := err.(*cerror.CError)

	s.True(ok)
	s.Equal(cerror.KindBadValidation.String(), cErr.Kind().String())
	s.Contains(cErr.Error(), "expected to have at least 1 elements")

	_, err = s.uc.UpdatePatient(ctx, id, &fhirModel.Parameters{
		Parameter: []*fhirModel.ParametersParameter{
			{
				Resource: map[string]interface{}{},
			},
		},
	})
	s.Error(err)

	cErr, ok = err.(*cerror.CError)

	s.True(ok)
	s.Equal(cerror.KindBadValidation.String(), cErr.Kind().String())
	s.Contains(cErr.Error(), "at least one of the parameters")

	p := new(fhirModel.Parameters)
	err = json.Unmarshal([]byte(updatePatientReqBody), p)
	s.NoError(err)

	_, err = s.uc.UpdatePatient(ctx, id, p)

	cErr, ok = err.(*cerror.CError)

	s.True(ok)
	s.Equal(cerror.KindBadValidation.String(), cErr.Kind().String())
	s.Contains(cErr.Error(), "url id and patient id")

	id = fhirModel.ID("9e293127-8ffc-462c-aea0-d5464794b526")

	s.fhir.patients = []*fhirModel.Patient{
		{
			DomainResource: fhirModel.DomainResource{
				Resource: fhirModel.Resource{
					ID: id,
				},
			},
			DeceasedBoolean: converto.BoolPointer(true),
		},
	}
	_, err = s.uc.UpdatePatient(ctx, id, p)
	s.Error(err)

	cErr, ok = err.(*cerror.CError)

	s.True(ok)
	s.Equal(cerror.KindBadValidation.String(), cErr.Kind().String())
	s.Contains(cErr.Error(), "inactive status")

	s.fhir.patients = []*fhirModel.Patient{
		{
			DomainResource: fhirModel.DomainResource{
				Resource: fhirModel.Resource{
					ID: id,
				},
			},
			Active:          converto.BoolPointer(true),
			DeceasedBoolean: converto.BoolPointer(true),
		},
	}

	_, err = s.uc.UpdatePatient(ctx, id, p)
	s.Error(err)

	cErr, ok = err.(*cerror.CError)

	s.True(ok)
	s.Equal(cerror.KindBadValidation.String(), cErr.Kind().String())
	s.Contains(cErr.Error(), "deceased status")

	s.fhir.patients = []*fhirModel.Patient{
		{
			DomainResource: fhirModel.DomainResource{
				Resource: fhirModel.Resource{
					ID: id,
				},
			},
			Active:          converto.BoolPointer(true),
			DeceasedBoolean: converto.BoolPointer(false),
		},
	}

	task, err := s.uc.UpdatePatient(ctx, id, p)
	s.NoError(err)
	s.Equal(fhirModel.TaskStatusCompleted, task.Status)
	s.Equal(fhirModel.TaskIntentOrder, task.Intent)
	s.Equal(1, len(task.Input))
	s.Equal(fmt.Sprintf("%s/%s", fhirModel.ResourceParameters, id), task.Input[0].ValueReference.Reference)
}

func (s *useCaseTestSuite) TestUpdatePatientIdentitySuccess() {
	p := new(fhirModel.Parameters)
	err := json.Unmarshal([]byte(updatePatientIdentityReqBody), p)
	s.NoError(err)

	id := fhirModel.ID("244a8e88-c0b0-4d60-b5d7-14afbe79f5f5")
	s.fhir.patients = append(s.fhir.patients, &fhirModel.Patient{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{
				ID: id,
			},
		},
		Active: converto.BoolPointer(true),
	})

	patient := new(fhirModel.Patient)
	interfaceToStruct(p.Parameter[1].Resource, patient)

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	patient.Identifier[0].Period.End = (*fhirModel.DateTime)(converto.TimePointer(today))
	p.Parameter[1].Resource = patient

	t, err := s.uc.UpdatePatientIdentity(context.Background(), id, p)

	s.NoError(err)
	s.NotNil(t)

	s.Equal(1, s.fhir.validateParametersCallsCount)
	s.Equal(1, len(s.fhir.bundles))
	s.Equal(2, len(s.fhir.bundles[0].Entry))

	bundleParameters, ok := s.fhir.bundles[0].Entry[0].Resource.(*fhirModel.Parameters)
	s.True(ok)
	s.Equal(p, bundleParameters)

	bundleTask, ok := s.fhir.bundles[0].Entry[1].Resource.(*fhirModel.Task)
	s.True(ok)
	s.Equal(t, bundleTask)

	//nolint:dupl
	expectedTask := &fhirModel.Task{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{
				ID:           t.ID,
				ResourceType: fhirModel.ResourceTask,
				Meta:         &fhirModel.Meta{Profile: []string{fhirModel.StructureDefinitionTaskPatientUpdateIdentity}},
			},
		},
		Status:         fhirModel.TaskStatusInProgress,
		BusinessStatus: &fhirModel.CodeableConcept{Text: entity.TaskBusinessStatusPatientIdentityUpdated},
		Intent:         fhirModel.TaskIntentOrder,
		AuthoredOn:     t.AuthoredOn,
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

	s.Equal(expectedTask, t)
	s.False(t.AuthoredOn.Time().IsZero())
	s.NotEmpty(t.ID)
}

func (s *useCaseTestSuite) TestUpdatePatientIdentityInternalValidate() {
	ctx := context.Background()
	id := fhirModel.ID(uuid.NewV4().String())
	s.fhir.patients = append(s.fhir.patients, &fhirModel.Patient{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{
				ID: fhirModel.ID("244a8e88-c0b0-4d60-b5d7-14afbe79f5f5"),
			},
		},
		DeceasedBoolean: converto.BoolPointer(true),
	})
	p := new(fhirModel.Parameters)
	err := json.Unmarshal([]byte(updatePatientIdentityWithoutParamsReqBody), p)
	s.NoError(err)

	_, err = s.uc.UpdatePatientIdentity(ctx, id, p)
	s.Error(err)
	s.Contains(err.Error(), "patient or confirmation method parameter")

	p.Parameter = append(p.Parameter, &fhirModel.ParametersParameter{
		Name: "confirmationMethod",
		ValueX: fhirModel.ValueX{
			ValueString: converto.StringPointer("+380672200333"),
		},
	})

	p = new(fhirModel.Parameters)
	err = json.Unmarshal([]byte(updatePatientIdentityReqBody), p)
	s.NoError(err)

	patient := new(fhirModel.Patient)
	interfaceToStruct(p.Parameter[1].Resource, patient)

	_, err = s.uc.UpdatePatientIdentity(ctx, id, p)
	s.Error(err)
	s.Contains(err.Error(), "url id and patient id")

	id = fhirModel.ID("244a8e88-c0b0-4d60-b5d7-14afbe79f5f5")

	_, err = s.uc.UpdatePatientIdentity(ctx, id, p)
	s.Error(err)
	s.Contains(err.Error(), "inactive status")

	s.fhir.patients[0].Active = converto.BoolPointer(true)

	_, err = s.uc.UpdatePatientIdentity(ctx, id, p)
	s.Error(err)
	s.Contains(err.Error(), "deceased status")

	s.fhir.patients[0].DeceasedBoolean = converto.BoolPointer(false)

	_, err = s.uc.UpdatePatientIdentity(ctx, id, p)
	s.Error(err)
	s.Contains(err.Error(), "identifier is expired")

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	patient.Identifier[0].Period.End = (*fhirModel.DateTime)(converto.TimePointer(today))
	patient.Extension[0].Extension[0].ValueCodeableConcept.Codings[0].Code = "AA"
	patient.Identifier[0].Type.Codings[0].Code = "AA"
	p.Parameter[1].Resource = patient

	_, err = s.uc.UpdatePatientIdentity(ctx, id, p)
	s.Error(err)
	s.Contains(err.Error(), "identifier code should be one of PRC,BN,DP,JHN,GCC,VS,PPN")

	patient.Extension[0].Extension[0].ValueCodeableConcept.Codings[0].Code = "SA"
	p.Parameter[1].Resource = patient

	_, err = s.uc.UpdatePatientIdentity(ctx, id, p)
	s.Error(err)
	s.Contains(err.Error(), "identifier code should be one of NI,DP,CZ,JHN")
}

func (s *useCaseTestSuite) TestUpdatePatientIdentitySearchDuplTasks() {
	ctx := context.Background()
	id := fhirModel.ID("244a8e88-c0b0-4d60-b5d7-14afbe79f5f5")
	s.fhir.patients = append(s.fhir.patients, &fhirModel.Patient{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{
				ID: id,
			},
		},
		Active: converto.BoolPointer(true),
	})

	p := new(fhirModel.Parameters)
	err := json.Unmarshal([]byte(updatePatientIdentityReqBody), p)
	s.NoError(err)

	patient := new(fhirModel.Patient)
	interfaceToStruct(p.Parameter[1].Resource, patient)

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	patient.Identifier[0].Period.End = (*fhirModel.DateTime)(converto.TimePointer(today))
	p.Parameter[1].Resource = patient

	_, err = s.uc.UpdatePatientIdentity(ctx, id, p)
	s.NoError(err)
	s.Equal(1, len(s.fhir.bundles))
	s.Equal(1, len(s.fhir.searchTaskByParamsArgs))
	s.Equal(&entity.SearchTaskByIdentifierParams{
		Value: "1058529940",
		Type:  "http://terminology.hl7.org/CodeSystem/v2-0203|NI",
	}, s.fhir.searchTaskByParamsArgs[0].Identifier)
	s.Nil(s.fhir.searchTaskByParamsArgs[0].Telecom)

	s.fhir.tasks = []*fhirModel.Task{new(fhirModel.Task)}
	_, err = s.uc.UpdatePatientIdentity(ctx, id, p)
	s.NoError(err)
	s.Equal(2, len(s.fhir.bundles))
	s.Equal(2, len(s.fhir.searchTaskByParamsArgs))

	s.fhir.tasks[0].ID = fhirModel.ID(uuid.NewV4().String())
	s.fhir.tasks[0].Status = fhirModel.TaskStatusInProgress
	_, err = s.uc.UpdatePatientIdentity(ctx, id, p)
	s.NoError(err)
	s.Equal(3, len(s.fhir.bundles))
	s.Equal(3, len(s.fhir.searchTaskByParamsArgs))
	s.NotNil(s.fhir.searchTaskByParamsArgs[2].Identifier)
	s.Equal(3, len(s.fhir.bundles[2].Entry))

	t, ok := s.fhir.bundles[2].Entry[2].Resource.(*fhirModel.Task)

	s.True(ok)
	s.Equal(s.fhir.tasks[0].ID, t.ID)
	s.Equal(fhirModel.TaskStatusCanceled, t.Status)
}

func (s *useCaseTestSuite) TestUpdatePatientIdentityGenerateOTP() {
	ctx := context.Background()
	id := fhirModel.ID("244a8e88-c0b0-4d60-b5d7-14afbe79f5f5")
	s.fhir.patients = append(s.fhir.patients, &fhirModel.Patient{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{
				ID: id,
			},
		},
		Active: converto.BoolPointer(true),
	})

	p := new(fhirModel.Parameters)
	err := json.Unmarshal([]byte(updatePatientIdentityReqBody), p)
	s.NoError(err)

	patient := new(fhirModel.Patient)
	interfaceToStruct(p.Parameter[1].Resource, patient)

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	patient.Identifier[0].Period.End = (*fhirModel.DateTime)(converto.TimePointer(today))
	p.Parameter[1].Resource = patient

	_, err = s.uc.UpdatePatientIdentity(ctx, id, p)
	s.NoError(err)

	s.Equal(1, len(s.otp.otps))
}

func (s *useCaseTestSuite) TestUpdatePatientIdentifierSearchDuplPatients() {
	ctx := context.Background()
	id := fhirModel.ID("244a8e88-c0b0-4d60-b5d7-14afbe79f5f5")
	s.fhir.patients = append(s.fhir.patients, &fhirModel.Patient{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{
				ID: id,
			},
		},
		Active: converto.BoolPointer(true),
	})

	p := new(fhirModel.Parameters)
	err := json.Unmarshal([]byte(updatePatientIdentityReqBody), p)
	s.NoError(err)

	patient := new(fhirModel.Patient)
	interfaceToStruct(p.Parameter[1].Resource, patient)

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	patient.Identifier[0].Period.End = (*fhirModel.DateTime)(converto.TimePointer(today))
	p.Parameter[1].Resource = patient

	_, err = s.uc.UpdatePatientIdentity(ctx, id, p)
	s.NoError(err)
	s.Equal(1, len(s.fhir.searchPatientByParamsArgs))
	s.Equal(&entity.SearchPatientByIdentifierParams{
		Value: patient.Identifier[0].Value,
		Type: fmt.Sprintf("%s|%s",
			patient.Identifier[0].Type.Codings[0].System,
			patient.Identifier[0].Type.Codings[0].Code),
	}, s.fhir.searchPatientByParamsArgs[0].Identifier)

	s.fhir.duplPatients = []*fhirModel.Patient{
		{DomainResource: fhirModel.DomainResource{Resource: fhirModel.Resource{ID: id}}},
	}

	_, err = s.uc.UpdatePatientIdentity(ctx, id, p)
	s.Error(err)
	s.Contains(err.Error(), "such person already exists")
}

func (s *useCaseTestSuite) TestConfirmUpdatePatientIdentitySuccess() {
	id := fhirModel.ID("244a8e88-c0b0-4d60-b5d7-14afbe79f5f5")
	s.fhir.patients = append(s.fhir.patients, &fhirModel.Patient{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{
				ID:           id,
				ResourceType: "Patient",
				Meta: &fhirModel.Meta{
					Profile: []string{"http://ksa-ehealth.sa/fhir/StructureDefinition/ksa-ehealth-patient-operation-update-identity"},
				},
			},
		},
		Active: converto.BoolPointer(true),
	})

	p := new(fhirModel.Parameters)
	err := json.Unmarshal([]byte(confirmUpdatePatientIdentityReqBody), p)
	s.NoError(err)

	patientParams := new(fhirModel.Parameters)
	err = json.Unmarshal([]byte(updatePatientIdentityReqBody), patientParams)
	s.NoError(err)

	s.fhir.parameters = []*fhirModel.Parameters{patientParams}

	s.fhir.tasks = []*fhirModel.Task{{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{
				ID: fhirModel.ID("6dc1c86e-3b4e-4e97-860c-9196bd9aa412"),
			},
		},
		Status: fhirModel.TaskStatusInProgress,
		Input: []*fhirModel.TaskInput{
			{
				ValueX: fhirModel.ValueX{
					ValueReference: &fhirModel.Reference{
						Reference: "Parameters/" + patientParams.ID.String(),
					},
				},
			},
		},
	}}
	s.otp.otps = map[string]*entity.OTP{
		"6dc1c86e-3b4e-4e97-860c-9196bd9aa412": {Code: "1234", Value: "+380672200333"},
	}

	actualTask, err := s.uc.ConfirmUpdatePatientIdentity(context.Background(), id, p)

	s.NoError(err)
	s.NotNil(actualTask)

	s.Equal(1, s.fhir.validateParametersCallsCount)
	s.Equal(1, len(s.otp.otps))
	s.Equal(1, len(s.fhir.bundles))
	s.Equal(3, len(s.fhir.bundles[0].Entry))

	bundleParameters, ok := s.fhir.bundles[0].Entry[0].Resource.(*fhirModel.Parameters)
	s.True(ok)
	s.Equal(p, bundleParameters)

	s.Equal(http.MethodPut, s.fhir.bundles[0].Entry[2].Request.Method)
	s.Equal("Task/6dc1c86e-3b4e-4e97-860c-9196bd9aa412", s.fhir.bundles[0].Entry[2].Request.URL)

	bundleTask, ok := s.fhir.bundles[0].Entry[2].Resource.(*fhirModel.Task)

	s.True(ok)
	s.Equal(fhirModel.TaskStatusCompleted, bundleTask.Status)
	s.Equal(&fhirModel.CodeableConcept{
		Text: entity.TaskBusinessStatusConfirmPatientIdentityUpdated,
	}, bundleTask.BusinessStatus)
	s.Equal(2, len(bundleTask.Input))
	s.Equal("Parameters/"+p.ID.String(), bundleTask.Input[1].ValueReference.Reference)
	s.Equal(1, len(bundleTask.Output))
	s.Equal("Patient/"+patientParams.ID.String(), bundleTask.Output[0].ValueReference.Reference)

	s.Equal(bundleTask, actualTask)

	bundlePatient, ok := s.fhir.bundles[0].Entry[1].Resource.(*fhirModel.Patient)
	s.True(ok)

	patient := new(fhirModel.Patient)
	interfaceToStruct(patientParams.Parameter[1].Resource, patient)
	s.NoError(err)

	patient.Active = converto.BoolPointer(true)
	s.Equal(patient, bundlePatient)
}

func (s *useCaseTestSuite) TestConfirmUpdatePatientIdentityInvalidParams() {
	id := fhirModel.ID("244a8e88-c0b0-4d60-b5d7-14afbe79f5f5")
	s.fhir.patients = append(s.fhir.patients, &fhirModel.Patient{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{
				ID: id,
			},
		},
		Active: converto.BoolPointer(true),
	})

	p := new(fhirModel.Parameters)
	err := json.Unmarshal([]byte(confirmUpdatePatientIdentityReqBody), p)
	s.NoError(err)

	otpParam := p.Parameter[0]
	taskParam := p.Parameter[1]

	p.Parameter = []*fhirModel.ParametersParameter{}

	_, err = s.uc.ConfirmUpdatePatientIdentity(context.Background(), id, p)

	s.Error(err)

	cErr, ok := err.(*cerror.CError)
	s.True(ok)
	s.Contains(fmt.Sprintf("%v", cErr.Payload()), "missing otp parameter")

	p.Parameter = []*fhirModel.ParametersParameter{otpParam}

	_, err = s.uc.ConfirmUpdatePatientIdentity(context.Background(), id, p)

	cErr, ok = err.(*cerror.CError)
	s.True(ok)
	s.Contains(fmt.Sprintf("%v", cErr.Payload()), "missing task_id parameter")

	p.Parameter = []*fhirModel.ParametersParameter{otpParam, taskParam}

	_, err = s.uc.ConfirmUpdatePatientIdentity(context.Background(), id, p)

	cErr, ok = err.(*cerror.CError)
	s.True(ok)
	s.NotContains(fmt.Sprintf("%v", cErr.Payload()), "missing otp parameter")
	s.NotContains(fmt.Sprintf("%v", cErr.Payload()), "missing correct task_id parameter")
}

func (s *useCaseTestSuite) TestConfirmUpdatePatientIdentityValidationErrors() {
	ctx := context.Background()
	id := fhirModel.ID("244a8e88-c0b0-4d60-b5d7-14afbe79f5f5")
	s.fhir.patients = append(s.fhir.patients, &fhirModel.Patient{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{
				ID: id,
			},
		},
		DeceasedBoolean: converto.BoolPointer(true),
	})

	p := new(fhirModel.Parameters)
	err := json.Unmarshal([]byte(confirmUpdatePatientIdentityReqBody), p)
	s.NoError(err)

	_, err = s.uc.ConfirmUpdatePatientIdentity(ctx, id, p)

	cErr, ok := err.(*cerror.CError)
	s.True(ok)
	s.Equal(cerror.KindNotExist.String(), cErr.Kind().String())
	s.Contains(err.Error(), "such patient request does not exist")

	s.fhir.tasks = []*fhirModel.Task{{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{
				ID: fhirModel.ID("6dc1c86e-3b4e-4e97-860c-9196bd9aa412"),
			},
		},
	}}

	_, err = s.uc.ConfirmUpdatePatientIdentity(ctx, id, p)

	cErr, ok = err.(*cerror.CError)
	s.True(ok)
	s.Equal(cerror.KindBadValidation.String(), cErr.Kind().String())
	s.Contains(err.Error(), "such patient request is not active")

	s.fhir.tasks[0].Status = fhirModel.TaskStatusInProgress

	_, err = s.uc.ConfirmUpdatePatientIdentity(ctx, id, p)

	cErr, ok = err.(*cerror.CError)
	s.True(ok)
	s.Equal(cerror.KindBadValidation.String(), cErr.Kind().String())
	s.Contains(err.Error(), "expected task input to have at least 1 elements")

	patientParams := new(fhirModel.Parameters)
	err = json.Unmarshal([]byte(updatePatientIdentityReqBody), patientParams)
	s.NoError(err)

	s.fhir.tasks[0].Input = []*fhirModel.TaskInput{
		{
			ValueX: fhirModel.ValueX{
				ValueReference: &fhirModel.Reference{
					Reference: "Parameters/" + patientParams.ID.String(),
				},
			},
		},
	}

	_, err = s.uc.ConfirmUpdatePatientIdentity(ctx, id, p)

	cErr, ok = err.(*cerror.CError)
	s.True(ok)
	s.Equal(cerror.KindNotExist.String(), cErr.Kind().String())
	s.Contains(err.Error(), "parameters for such patient request do not exist")

	patient := new(fhirModel.Patient)
	interfaceToStruct(patientParams.Parameter[1].Resource, patient)

	s.fhir.parameters = []*fhirModel.Parameters{patientParams}

	_, err = s.uc.ConfirmUpdatePatientIdentity(ctx, id, p)
	s.Error(err)
	s.Contains(err.Error(), "inactive status")

	s.fhir.patients[0].Active = converto.BoolPointer(true)

	_, err = s.uc.ConfirmUpdatePatientIdentity(ctx, id, p)
	s.Error(err)
	s.Contains(err.Error(), "deceased status")

	s.fhir.patients[0].DeceasedBoolean = converto.BoolPointer(false)

	_, err = s.uc.ConfirmUpdatePatientIdentity(ctx, id, p)

	cErr, ok = err.(*cerror.CError)
	s.True(ok)
	s.Equal(cerror.KindNotExist.String(), cErr.Kind().String())
	s.Contains(err.Error(), "otp for such patient request does not exist")

	s.otp.otps = map[string]*entity.OTP{
		s.fhir.tasks[0].ID.String(): {
			Code:  "1234",
			Value: converto.StringValue(patientParams.Parameter[0].ValueString),
		},
	}

	s.fhir.duplPatients = []*fhirModel.Patient{patient}
	_, err = s.uc.ConfirmUpdatePatientIdentity(ctx, id, p)

	s.Error(err)
	s.Contains(err.Error(), "such person already exists")
	s.Equal(1, len(s.fhir.searchPatientByParamsArgs))
	s.Equal(&entity.SearchPatientParams{
		Identifier: &entity.SearchPatientByIdentifierParams{
			Value: patient.Identifier[0].Value,
			Type: fmt.Sprintf(
				"%s|%s",
				patient.Identifier[0].Type.Codings[0].System,
				patient.Identifier[0].Type.Codings[0].Code),
		},
	}, s.fhir.searchPatientByParamsArgs[0])

	s.Equal(1, len(s.fhir.bundles))
	s.Equal(2, len(s.fhir.bundles[0].Entry))

	t, ok := s.fhir.bundles[0].Entry[1].Resource.(*fhirModel.Task)

	s.True(ok)
	s.Equal(fhirModel.TaskStatusRejected, t.Status)

	s.fhir.duplPatients = nil
	s.fhir.searchPatientByParamsArgs = nil

	s.fhir.tasks = []*fhirModel.Task{{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{
				ID: fhirModel.ID("6dc1c86e-3b4e-4e97-860c-9196bd9aa412"),
			},
		},
		Status: fhirModel.TaskStatusInProgress,
		Input: []*fhirModel.TaskInput{
			{
				ValueX: fhirModel.ValueX{
					ValueReference: &fhirModel.Reference{
						Reference: "Parameters/" + patientParams.ID.String(),
					},
				},
			},
		},
	}}

	_, err = s.uc.ConfirmUpdatePatientIdentity(ctx, id, p)
	s.NoError(err)

	s.Equal(1, len(s.fhir.searchPatientByParamsArgs))
	s.Equal(&entity.SearchPatientByIdentifierParams{
		Value: patient.Identifier[0].Value,
		Type: fmt.Sprintf("%s|%s",
			patient.Identifier[0].Type.Codings[0].System,
			patient.Identifier[0].Type.Codings[0].Code),
	}, s.fhir.searchPatientByParamsArgs[0].Identifier)

	s.fhir.duplPatients = []*fhirModel.Patient{
		{DomainResource: fhirModel.DomainResource{Resource: fhirModel.Resource{ID: id}}},
	}
}

func (s *useCaseTestSuite) TestUpdateEmailPatient() {
	ctx := context.Background()
	id := fhirModel.ID(uuid.NewV4().String())

	_, err := s.uc.UpdatePatientEmail(ctx, id, &fhirModel.Parameters{})
	s.Error(err)

	cErr, ok := err.(*cerror.CError)

	s.True(ok)
	s.Equal(cerror.KindBadValidation.String(), cErr.Kind().String())
	s.Contains(cErr.Error(), "expected to have at least 1 elements")

	_, err = s.uc.UpdatePatient(ctx, id, &fhirModel.Parameters{
		Parameter: []*fhirModel.ParametersParameter{
			{
				Resource: map[string]interface{}{},
			},
		},
	})
	s.Error(err)

	cErr, ok = err.(*cerror.CError)

	s.True(ok)
	s.Equal(cerror.KindBadValidation.String(), cErr.Kind().String())
	s.Contains(cErr.Error(), "at least one of the parameters")

	p := new(fhirModel.Parameters)
	err = json.Unmarshal([]byte(updateEmailPatientReqBody), p)
	s.NoError(err)

	_, err = s.uc.UpdatePatientEmail(ctx, id, p)

	cErr, ok = err.(*cerror.CError)

	s.True(ok)
	s.Equal(cerror.KindBadValidation.String(), cErr.Kind().String())
	s.Contains(cErr.Error(), "input id and patient id")

	id = fhirModel.ID("9e293127-8ffc-462c-aea0-d5464794b527")

	s.fhir.patients = []*fhirModel.Patient{
		{
			DomainResource: fhirModel.DomainResource{
				Resource: fhirModel.Resource{
					ID: id,
				},
			},
			DeceasedBoolean: converto.BoolPointer(true),
		},
	}
	_, err = s.uc.UpdatePatientEmail(ctx, id, p)
	s.Error(err)

	cErr, ok = err.(*cerror.CError)

	s.True(ok)
	s.Equal(cerror.KindBadValidation.String(), cErr.Kind().String())
	s.Contains(cErr.Error(), "inactive status")

	s.fhir.patients = []*fhirModel.Patient{
		{
			DomainResource: fhirModel.DomainResource{
				Resource: fhirModel.Resource{
					ID: id,
				},
			},
			Active:          converto.BoolPointer(true),
			DeceasedBoolean: converto.BoolPointer(true),
		},
	}

	_, err = s.uc.UpdatePatientEmail(ctx, id, p)
	s.Error(err)

	cErr, ok = err.(*cerror.CError)

	s.True(ok)
	s.Equal(cerror.KindBadValidation.String(), cErr.Kind().String())
	s.Contains(cErr.Error(), "deceased status")

	var t []*fhirModel.ContactPoint
	t = append(t, &fhirModel.ContactPoint{
		System: "email",
		Value:  "test@test.com",
	})

	s.fhir.patients = []*fhirModel.Patient{
		{
			DomainResource: fhirModel.DomainResource{
				Resource: fhirModel.Resource{
					ID: id,
				},
			},
			Active:          converto.BoolPointer(true),
			DeceasedBoolean: converto.BoolPointer(false),
			Telecom:         t,
		},
	}

	task, err := s.uc.UpdatePatientEmail(ctx, id, p)
	s.NoError(err)
	s.Equal("completed", task.Status)
}

func mapFromStruct(src interface{}) map[string]interface{} {
	b, _ := json.Marshal(src)
	m := make(map[string]interface{})
	_ = json.Unmarshal(b, &m)

	return m
}

func interfaceToStruct(i, dst interface{}) {
	b, _ := json.Marshal(i)
	if err := json.Unmarshal(b, dst); err != nil {
		panic(err)
	}
}

var (
	createPatientReqBody = `{
		"resourceType": "Parameters",
		"id": "9e293127-8ffc-462c-aea0-d5464794b526",
		"meta": {
			"profile": [
				"http://ksa-ehealth.sa/fhir/StructureDefinition/ksa-ehealth-parameters-patient-create-request"
			]
		},
		"parameter": [
			{
				"name": "patient",
				"resource": {
					"resourceType": "Patient",
					"id": "9e293127-8ffc-462c-aea0-d5464794b526",
					"meta": {
						"profile": [
							"http://ksa-ehealth.sa/fhir/StructureDefinition/ksa-ehealth-patient-identified"
						]
					},
					"text": {
						"status": "generated",
						"div": "<div>!-- Snipped for Brevity --></div>"
					},
					"extension":[
			   			{
			   			   "url":"http://ksa-ehealth.sa/fhir/StructureDefinition/ksa-ehealth-patient-nationality",
			   			   "extension":[
			   				  	{
			   						"url":"code",
			   						"valueCodeableConcept":{
			   						   "coding":[
			   							  {
			   								 "code":"SA",
			   								 "system":"urn:iso:std:iso:3166:-2",
			   								 "display":"Saudi, Saudi Arabian"
			   							  }
			   						   ]
			   						}
			   					}
							]
						}
					],
					"identifier": [
						{
							"type": {
								"coding": [
									{
										"system": "http://terminology.hl7.org/CodeSystem/v2-0203",
										"code": "NI"
									}
								]
							},
							"system": "urn:oid:1.2.36.146.595.217.0.1",
							"value": "1825641265",
							"period": {
								"start": "2020-05-06",
								"end":"2050-01-01"
							},
							"assigner": {
								"display": "Acme Healthcare"
							}
						}
					],
					"active": true,
					"name": [
						{
							"extension": [
								{
									"url": "http://nphies.sa/fhir/ksa/nphies-fs/StructureDefinition/wasfaty-extension-humanname-language",
									"valueCode": "en"
								}
							],
							"use": "official",
							"family": "Chalmers",
							"given": [
								"Peter",
								"James"
							]
						},
						{
							"extension": [
								{
									"url": "http://nphies.sa/fhir/ksa/nphies-fs/StructureDefinition/wasfaty-extension-humanname-language",
									"valueCode": "ar"
								}
							],
							"use": "official",
							"family": "Chalmers",
							"given": [
								"Peter",
								"James"
							]
						}
					],
					"telecom": [
						{
							"system": "phone",
							"value": "+380673212121",
							"use": "mobile",
							"rank": 2
						}
					],
	                "identifier": [
	                    {
	                        "type": {
	                            "coding": [
	                                {
	                                    "system": "http://terminology.hl7.org/CodeSystem/v2-0203",
	                                    "code": "NI"
	                                }
	                            ]
	                        },
	                        "system": "urn:oid:1.2.36.146.595.217.0.1",
	                        "value": "1058529940",
	                        "period": {
	                            "start": "2020-05-06",
	                            "end":"2050-01-01"
	                        },
	                        "assigner": {
	                            "display": "Acme Healthcare"
	                        }
	                    }
	                ],
	                "active": true,
	                "name": [
	                    {
	                        "extension": [
	                            {
	                                "url": "http://nphies.sa/fhir/ksa/nphies-fs/StructureDefinition/wasfaty-extension-humanname-language",
	                                "valueCode": "en"
	                            }
	                        ],
	                        "use": "official",
	                        "family": "Chalmers",
	                        "given": [
	                            "Peter",
	                            "James"
	                        ]
	                    },
	                    {
	                        "extension": [
	                            {
	                                "url": "http://nphies.sa/fhir/ksa/nphies-fs/StructureDefinition/wasfaty-extension-humanname-language",
	                                "valueCode": "ar"
	                            }
	                        ],
	                        "use": "official",
	                        "family": "Chalmers",
	                        "given": [
	                            "Peter",
	                            "James"
	                        ]
	                    }
	                ],
	                "telecom": [
	                    {
	                        "system": "phone",
	                        "value": "+380673212121",
	                        "use": "mobile",
	                        "rank": 2
	                    }
	                ],
	                "gender": "male",
	                "birthDate": "2020-12-25",
	                "maritalStatus": {
	                    "coding": [
	                        {
	                            "system": "http://terminology.hl7.org/CodeSystem/v3-MaritalStatus",
	                            "code": "M"
	                        }
	                    ]
	                },
	                "address": [
	                    {
	                        "use": "home",
	                        "type": "both",
	                        "text": "534 Erewhon St PeasantVille, Rainbow, Vic  3999",
	                        "line": [
	                            "534 Erewhon St"
	                        ],
	                        "city": "PleasantVille",
	                        "district": "Rainbow",
	                        "state": "Vic",
	                        "postalCode": "3999",
	                        "period": {
	                            "start": "1974-12-25"
	                        }
	                    }
	                ]
	            }
	        }
	    ]
	}`

	updatePatientReqBody = `{
		"resourceType": "Parameters",
		"id": "9e293127-8ffc-462c-aea0-d5464794b526",
		"meta": {
			"profile": [
				"http://ksa-ehealth.sa/fhir/StructureDefinition/ksa-ehealth-parameters-patient-update"
			]
		},
		"parameter": [
			{
				"name": "patient",
				"resource": {
					"resourceType": "Patient",
					"id": "9e293127-8ffc-462c-aea0-d5464794b526",
					"meta": {
						"profile": [
							"http://ksa-ehealth.sa/fhir/StructureDefinition/ksa-ehealth-patient-operation-update"
						]
					},
					"extension": [
					{
						"url": "http://nphies.sa/fhir/ksa/nphies-fs/StructureDefinition/extension-patient-religion",
						"valueCodeableConcept": {
							"coding": [
									{
										"code": "code",
										"system": "system",
										"version": "version",
										"display": "display"
									}
								],
							"text": "text"
						}
					},
					{
						"url": "http://nphies.sa/fhir/ksa/nphies-fs/StructureDefinition/wasfaty-extension-patient-importance",
						"valueCodeableConcept": {
							"coding": [
									{
										"code": "code",
										"system": "system",
										"version": "version",
										"display": "display"
									}
								],
							"text": "text"
						}
					},
					{
						"url": "http://nphies.sa/fhir/ksa/nphies-fs/StructureDefinition/wasfaty-extension-patient-occupation",
						"valueCodeableConcept": {
							"coding": [
									{
										"code": "code",
										"system": "system",
										"version": "version",
										"display": "display"
									}
								],
							"text": "text"
						}
					},
					{
						"url": "http://wasfaty.sa/fhir/StructureDefinition/wasfaty-extension-patient-citizenship",
						"valueCodeableConcept": {
							"coding": [
									{
										"code": "code",
										"system": "system",
										"version": "version",
										"display": "display"
									}
								],
							"text": "text"
						}
					},
					{
						"url": "http://wasfaty.sa/fhir/StructureDefinition/wasfaty-extension-patient-nationality",
						"valueCodeableConcept": {
							"coding": [
									{
										"code": "code",
										"system": "system",
										"version": "version",
										"display": "display"
									}
								],
							"text": "text"
						}
					}],
					"contact": [
						{
							"id": "example",
							"relationship": [
								{
									"coding": [
										{
											"code": "code",
											"system": "system",
											"version": "version",
											"display": "display"
										}
									],
									"text": "text"
								}
							],
							"name": {
								"use": "use",
								"text": "text",
								"family": "family",
								"given": [
									"given1",
									"given2"
								],
								"prefix": [
									"prefix1",
									"prefix2"
								],
								"suffix": [
									"suffix1",
									"suffix2"
								]
							},
							"telecom": [
								{
									"system": "phone",
									"value": "(03) 3410 5613",
									"use": "mobile",
									"rank": 2
								}
							]
						}

					],
					"communication": [
						{
							"id": "example",
							"language": {
								"coding": [
									{
										"code": "code",
										"system": "system",
										"version": "version",
										"display": "display"
									}
								]
							},
							"preferred": false
						}
					],
					"maritalStatus": {
						"coding": [
							{
								"system": "http://terminology.hl7.org/CodeSystem/v3-MaritalStatus",
								"code": "M"
							}
						]
					}
				}
			}
		]
	}`

	updateEmailPatientReqBody = `{
		"resourceType": "Parameters",
		"id": "9e293127-8ffc-462c-aea0-d5464794b527",
		"meta": {
			"profile": [
				"http://ksa-ehealth.sa/fhir/StructureDefinition/ksa-ehealth-parameters-patient-update-email"
			]
		},
		"parameter": [
			{
				"name": "patient",
				"resource": {
					"resourceType": "Patient",
					"id": "9e293127-8ffc-462c-aea0-d5464794b527",
					"meta": {
						"profile": [
							"http://ksa-ehealth.sa/fhir/StructureDefinition/ksa-ehealth-patient-operation-update-email"
						]
					},
					"telecom": [
                    	{
                        	"system": "email",
                        	"value": "test@test1.test"
                    	}
                	] 
				}
			}
		]
	}`

	updatePatientIdentityWithoutParamsReqBody = `{
		"resourceType": "Parameters",
		"id": "244a8e88-c0b0-4d60-b5d7-14afbe79f5f5",
		"meta": {
			"profile": [
				"http://ksa-ehealth.sa/fhir/StructureDefinition/ksa-ehealth-parameters-patient-update-identity"
			]
		},
		"parameter": [
		]
	}`

	updatePatientIdentityReqBody = `{
		"resourceType": "Parameters",
		"id": "244a8e88-c0b0-4d60-b5d7-14afbe79f5f5",
		"meta": {
			"profile": [
				"http://ksa-ehealth.sa/fhir/StructureDefinition/ksa-ehealth-parameters-patient-update-identity"
			]
		},
		"parameter": [
			{
				"name": "confirmationMethod",
				"valueString": "+380672200333"
			},
			{
				"name": "patient",
				"resource": {
					"resourceType": "Patient",
					"id": "244a8e88-c0b0-4d60-b5d7-14afbe79f5f5",
					"meta": {
						"profile": [
							"http://ksa-ehealth.sa/fhir/StructureDefinition/ksa-ehealth-patient-operation-update-identity"
						]
					},
					"extension": [
						{
							"url": "http://ksa-ehealth.sa/fhir/StructureDefinition/ksa-ehealth-patient-nationality",
							"extension": [
								{
									"url": "code",
									"valueCodeableConcept": {
										"coding": [
											{
												"code": "SA",
												"system": "urn:iso:std:iso:3166:-2",
												"display": "Saudi, Saudi Arabian"
											}
										]
									}
								}
							]
						}
					],
					"identifier": [
						{
							"type": {
								"coding": [
									{
										"system": "http://terminology.hl7.org/CodeSystem/v2-0203",
										"code": "NI"
									}
								]
							},
							"system": "http://nphies.sa/identifier/passportnumber",
							"value": "1058529940",
							"period": {
								"start": "2022-02-15",
								"end": "2022-03-16"
							},
							"assigner": {
								"display": "Acme Healthcare"
							}
						}
					],
					"name": [
						{
							"extension": [
								{
									"url": "http://ksa-ehealth.sa/fhir/ksa/nphies-fs/StructureDefinition/ksa-ehealth-humanname-language",
									"valueCode": "en"
								}
							],
							"use": "official",
							"family": "AL-SAUD",
							"given": [
								"Ahmad",
								"Hussain",
								"Khan",
								"Manzoor"
							]
						},
						{
							"extension": [
								{
									"url": "http://ksa-ehealth.sa/fhir/ksa/nphies-fs/StructureDefinition/ksa-ehealth-humanname-language",
									"valueCode": "ar"
								}
							],
							"use": "official",
							"family": "آل سعود",
							"given": [
								"أحمد",
								"حسين",
								"خان",
								"منصور"
							]
						}
					],
					"gender": "male",
					"birthDate": "1992-10-02"
				}
			}
		]
	}`

	confirmCreatePatientReqBody = `{
		"resourceType": "Parameters",
		"id": "9e293127-8ffc-462c-aea0-d5464794b526",
		"meta": {
			"profile": [
				"http://ksa-ehealth.sa/fhir/StructureDefinition/ksa-ehealth-parameters-patient-confirm-request"
			]
		},
		"parameter": [
			{
				"name": "otp",
				"valueString": "2655"
			},
			{
				"name": "task_id",
				"valueReference": {
							"reference": "Task/9e293127-8ffc-462c-aea0-d5464794b526"
						}
			}
		]
	}`

	confirmUpdatePatientIdentityReqBody = `{
		"resourceType": "Parameters",
		"id": "b488aa02-f181-4b50-bdca-63b74c5ee447",
		"meta": {
			"profile": [
				"http://ksa-ehealth.sa/fhir/StructureDefinition/ksa-ehealth-parameters-patient-confirm-identity"
			]
		},
		"parameter": [
			{
				"name": "otp",
				"valueString": "1234"
			},
			{
				"name": "task_id",
				"valueReference": {
							"reference": "Task/6dc1c86e-3b4e-4e97-860c-9196bd9aa412"
				}
			}
		]
	}`
)
