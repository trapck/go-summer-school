package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/suite"
	"wasfaty.api/pkg/cerror"
	"wasfaty.api/pkg/converto"
	"wasfaty.api/pkg/env"
	"wasfaty.api/pkg/fhir/ferror"
	fhirModel "wasfaty.api/pkg/fhir/model"
	pkgfiber "wasfaty.api/pkg/http/fiber"
	"wasfaty.api/pkg/log"
	"wasfaty.api/services/mpi/controller/http"
)

type testUseCase struct {
	createPatientFunc                func(ctx context.Context, p *fhirModel.Parameters) (*fhirModel.Task, error)
	updatePatientFunc                func(ctx context.Context, id fhirModel.ID, p *fhirModel.Parameters) (*fhirModel.Task, error)
	updateEmailPatientFunc           func(ctx context.Context, id fhirModel.ID, p *fhirModel.Parameters) (*fhirModel.Task, error)
	confirmCreatePatientFunc         func(ctx context.Context, p *fhirModel.Parameters) (*fhirModel.Task, error)
	updatePatientIdentityFunc        func(ctx context.Context, id fhirModel.ID, p *fhirModel.Parameters) (*fhirModel.Task, error)
	confirmUpdatePatientIdentityFunc func(ctx context.Context, id fhirModel.ID, p *fhirModel.Parameters) (*fhirModel.Task, error)
}

func (tuc *testUseCase) CreatePatient(ctx context.Context, p *fhirModel.Parameters) (*fhirModel.Task, error) {
	return tuc.createPatientFunc(ctx, p)
}

func (tuc *testUseCase) UpdatePatient(ctx context.Context, id fhirModel.ID, p *fhirModel.Parameters) (*fhirModel.Task, error) {
	return tuc.updatePatientFunc(ctx, id, p)
}

func (tuc *testUseCase) UpdatePatientEmail(ctx context.Context, id fhirModel.ID, p *fhirModel.Parameters) (*fhirModel.Task, error) {
	return tuc.updateEmailPatientFunc(ctx, id, p)
}

func (tuc *testUseCase) ConfirmCreatePatient(ctx context.Context, p *fhirModel.Parameters) (*fhirModel.Task, error) {
	return tuc.confirmCreatePatientFunc(ctx, p)
}

func (tuc *testUseCase) UpdatePatientIdentity(ctx context.Context, id fhirModel.ID, p *fhirModel.Parameters) (
	*fhirModel.Task, error) {
	return tuc.updatePatientIdentityFunc(ctx, id, p)
}

func (tuc *testUseCase) ConfirmUpdatePatientIdentity(ctx context.Context, id fhirModel.ID, p *fhirModel.Parameters) (
	*fhirModel.Task, error) {
	return tuc.confirmUpdatePatientIdentityFunc(ctx, id, p)
}

type handlerTestSuite struct {
	suite.Suite
	uc *testUseCase
	s  *pkgfiber.Server
}

func TestHandlerTestSuite(t *testing.T) {
	log.SetGlobalLogLevel("fatal")
	suite.Run(t, new(handlerTestSuite))
}

func (s *handlerTestSuite) SetupSuite() {
	s.uc = new(testUseCase)
	s.s = http.NewServer(env.HTTPServer{}, env.Service{}, nil, s.uc)
}

func (s *handlerTestSuite) TearDownTest() {
	s.uc.createPatientFunc = nil
	s.uc.updatePatientFunc = nil
	s.uc.updateEmailPatientFunc = nil
	s.uc.confirmCreatePatientFunc = nil
	s.uc.updatePatientIdentityFunc = nil
	s.uc.confirmUpdatePatientIdentityFunc = nil
}

func (s *handlerTestSuite) TearDownSuite() {

}

var (
	task = &fhirModel.Task{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{
				ID: fhirModel.ID("task-123"),
			},
		},
	}
)

func preparePatientReq(profile string, telecom []*fhirModel.ContactPoint) (*fhirModel.Patient, *fhirModel.Parameters) {
	p := &fhirModel.Patient{
		DomainResource: fhirModel.DomainResource{
			Resource: fhirModel.Resource{ID: fhirModel.ID("123")},
		},
	}

	if len(telecom) > 0 {
		p.Telecom = telecom
	}

	r := &fhirModel.Parameters{
		Resource: fhirModel.Resource{
			Meta: &fhirModel.Meta{Profile: []string{profile}},
		},
		Parameter: []*fhirModel.ParametersParameter{{Resource: p}},
	}

	return p, r
}

func prepareConfirmReq(profile string) *fhirModel.Parameters {
	r := &fhirModel.Parameters{
		Resource: fhirModel.Resource{
			Meta: &fhirModel.Meta{Profile: []string{profile}},
		},
		Parameter: []*fhirModel.ParametersParameter{
			{Name: "otp", ValueX: fhirModel.ValueX{ValueString: converto.StringPointer("1234")}},
			{Name: "task_id", ValueX: fhirModel.ValueX{ValueReference: &fhirModel.Reference{Reference: "Task/123"}}},
		},
	}

	return r
}

//nolint: dupl
func (s *handlerTestSuite) TestUpdatePatient() {
	var isCalled bool

	patient, req := preparePatientReq(fhirModel.StructureDefinitionPatientUpdateRequest, nil)

	s.uc.updatePatientFunc = func(ctx context.Context, id fhirModel.ID, p *fhirModel.Parameters) (*fhirModel.Task, error) {
		isCalled = true
		dst := new(fhirModel.Patient)

		interfaceToStruct(p.Parameter[0].Resource, dst)

		p.Parameter[0].Resource = dst

		s.Equal(patient.ID, id)
		s.Equal(req, p)

		return task, nil
	}

	tm := &testModel{
		method:       fiber.MethodPost,
		route:        fmt.Sprintf("/Patient/%s/$update", patient.ID),
		req:          req,
		dst:          new(fhirModel.Task),
		expectedCode: fiber.StatusOK,
		assertFn: func(code int, resp interface{}) {
			body, ok := resp.(*fhirModel.Task)
			s.True(ok)
			s.Equal(task, body)
		}}

	testByModel(s, tm)
	s.True(isCalled)

	s.testProfileError(req, tm)
}

func (s *handlerTestSuite) TestCreatePatient() {
	var isCalled bool

	_, req := preparePatientReq(fhirModel.StructureDefinitionPatientCreateRequest, nil)

	s.uc.createPatientFunc = func(ctx context.Context, p *fhirModel.Parameters) (*fhirModel.Task, error) {
		isCalled = true
		dst := new(fhirModel.Patient)

		interfaceToStruct(p.Parameter[0].Resource, dst)

		p.Parameter[0].Resource = dst

		s.Equal(req, p)

		return task, nil
	}

	tm := &testModel{
		method:       fiber.MethodPost,
		route:        "/Patient/$create-request",
		req:          req,
		dst:          new(fhirModel.Task),
		expectedCode: fiber.StatusOK,
		assertFn: func(code int, resp interface{}) {
			body, ok := resp.(*fhirModel.Task)
			s.True(ok)
			s.Equal(task, body)
		}}

	testByModel(s, tm)
	s.True(isCalled)

	s.testProfileError(req, tm)
}

//nolint: dupl
func (s *handlerTestSuite) TestUpdatePatientIdentity() {
	var isCalled bool

	patient, req := preparePatientReq(fhirModel.StructureDefinitionPatientUpdateIdentityRequest, nil)

	s.uc.updatePatientIdentityFunc = func(ctx context.Context, id fhirModel.ID, p *fhirModel.Parameters) (*fhirModel.Task, error) {
		isCalled = true
		dst := new(fhirModel.Patient)

		interfaceToStruct(p.Parameter[0].Resource, dst)

		p.Parameter[0].Resource = dst

		s.Equal(patient.ID, id)
		s.Equal(req, p)

		return task, nil
	}

	tm := &testModel{
		method:       fiber.MethodPost,
		route:        fmt.Sprintf("/Patient/%s/$update-identity", patient.ID),
		req:          req,
		dst:          new(fhirModel.Task),
		expectedCode: fiber.StatusOK,
		assertFn: func(code int, resp interface{}) {
			body, ok := resp.(*fhirModel.Task)
			s.True(ok)
			s.Equal(task, body)
		}}

	testByModel(s, tm)
	s.True(isCalled)

	s.testProfileError(req, tm)
}

func (s *handlerTestSuite) TestConfirmCreatePatient() {
	var isCalled bool

	req := prepareConfirmReq(fhirModel.StructureDefinitionPatientConfirmCreateRequest)

	s.uc.confirmCreatePatientFunc = func(ctx context.Context, p *fhirModel.Parameters) (*fhirModel.Task, error) {
		isCalled = true

		s.Equal(req, p)

		return task, nil
	}

	tm := &testModel{
		method:       fiber.MethodPost,
		route:        "/Patient/$confirm-request",
		req:          req,
		dst:          new(fhirModel.Task),
		expectedCode: fiber.StatusCreated,
		assertFn: func(code int, resp interface{}) {
			body, ok := resp.(*fhirModel.Task)
			s.True(ok)
			s.Equal(task, body)
		}}

	testByModel(s, tm)
	s.True(isCalled)

	s.testProfileError(req, tm)
}

func (s *handlerTestSuite) TestConfirmUpdatePatientIdentity() {
	var isCalled bool

	req := prepareConfirmReq(fhirModel.StructureDefinitionPatientConfirmUpdateIdentityRequest)
	patientID := fhirModel.ID("123")

	s.uc.confirmUpdatePatientIdentityFunc = func(ctx context.Context, id fhirModel.ID, p *fhirModel.Parameters) (*fhirModel.Task, error) {
		isCalled = true

		s.Equal(patientID, id)
		s.Equal(req, p)

		return task, nil
	}

	tm := &testModel{
		method:       fiber.MethodPost,
		route:        fmt.Sprintf("/Patient/%s/$confirm-identity", patientID),
		req:          req,
		dst:          new(fhirModel.Task),
		expectedCode: fiber.StatusOK,
		assertFn: func(code int, resp interface{}) {
			body, ok := resp.(*fhirModel.Task)
			s.True(ok)
			s.Equal(task, body)
		}}

	testByModel(s, tm)
	s.True(isCalled)

	s.testProfileError(req, tm)
}

func (s *handlerTestSuite) TestUpdateEmailPatient() {
	var isCalled bool

	var t []*fhirModel.ContactPoint
	t = append(t, &fhirModel.ContactPoint{
		System: "email",
		Value:  "test@test.com",
	})

	patient, req := preparePatientReq(fhirModel.StructureDefinitionPatientUpdateEmailRequest, t)

	s.uc.updateEmailPatientFunc = func(ctx context.Context, id fhirModel.ID, p *fhirModel.Parameters) (*fhirModel.Task, error) {
		isCalled = true
		dst := &fhirModel.Task{
			Status: "completed",
			DomainResource: fhirModel.DomainResource{
				Resource: fhirModel.Resource{
					Meta: &fhirModel.Meta{
						Profile: []string{fhirModel.StructureDefinitionTaskPatientUpdateEmail},
					},
				},
			},
		}

		s.Equal(patient.ID, id)
		s.Equal(req.Meta, p.Meta)

		return dst, nil
	}

	tm := &testModel{
		method:       fiber.MethodPost,
		route:        fmt.Sprintf("/Patient/%s/$update-email", patient.ID),
		req:          req,
		dst:          new(fhirModel.Task),
		expectedCode: fiber.StatusOK,
		assertFn: func(code int, resp interface{}) {
			body, ok := resp.(*fhirModel.Task)
			s.True(ok)
			s.Equal("completed", body.Status)
			s.Equal([]string{fhirModel.StructureDefinitionTaskPatientUpdateEmail}, body.Meta.Profile)
		}}

	testByModel(s, tm)
	s.True(isCalled)

	s.testProfileError(req, tm)
}

type testModel struct {
	method       string
	route        string
	req          interface{}
	dst          interface{}
	expectedCode int
	assertFn     func(code int, respBody interface{})
}

func testByModel(s *handlerTestSuite, m *testModel) {
	code, err := s.makeReq(m.method, m.route, m.req, m.dst)
	s.NoError(err)
	s.Equal(m.expectedCode, code)

	if m.assertFn != nil {
		m.assertFn(code, m.dst)
	}
}

func (s *handlerTestSuite) makeReq(method, route string, body, dst interface{}) (status int, err error) {
	buf := new(bytes.Buffer)

	if body != nil {
		b, _ := json.Marshal(body)
		_, _ = buf.Write(b)
	}

	req := httptest.NewRequest(method, route, buf)
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

	resp, err := s.s.Fiber().Test(req)

	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		if resp != nil {
			return resp.StatusCode, err
		}

		return cerror.ErrKind(err).HTTPCode(), err
	}

	if dst != nil {
		b, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(b, dst)
	}

	return resp.StatusCode, nil
}

func (s *handlerTestSuite) testProfileError(req *fhirModel.Parameters, tm *testModel) {
	req.Meta.Profile[0] += "123"
	tm.expectedCode = fiber.StatusUnprocessableEntity
	tm.dst = new(fhirModel.OperationOutcome)
	tm.assertFn = s.assertErr

	testByModel(s, tm)
}

func (s *handlerTestSuite) assertErr(code int, resp interface{}) {
	body, ok := resp.(*fhirModel.OperationOutcome)
	s.True(ok)
	s.NotEmpty(body.Issue[0])
	s.Equal(body.Issue[0].Code, ferror.OutcomeCodeFromHTTP(code))
}

func interfaceToStruct(i, dst interface{}) {
	b, _ := json.Marshal(i)
	_ = json.Unmarshal(b, dst)
}
