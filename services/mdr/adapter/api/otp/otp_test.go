package otp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/valyala/fasthttp"
	"wasfaty.api/pkg/cerror"
	"wasfaty.api/pkg/env"
	"wasfaty.api/pkg/http/consts"
	"wasfaty.api/pkg/log"
	"wasfaty.api/services/mpi/adapter/api/otp"
	"wasfaty.api/services/mpi/entity"
)

type otpTestSuite struct {
	suite.Suite
	cfg        *env.OTPClient
	otpClient  *otp.Client
	httpClient mockClient
}

type mockClient struct {
	DoFunc func(req *fasthttp.Request, resp *fasthttp.Response) error
}

func (m *mockClient) Do(req *fasthttp.Request, resp *fasthttp.Response) error {
	return m.DoFunc(req, resp)
}

func (m *mockClient) DoTimeout(req *fasthttp.Request, resp *fasthttp.Response, timeout time.Duration) error {
	return m.DoFunc(req, resp)
}

func TestOTPAdapterTestSuite(t *testing.T) {
	log.SetGlobalLogLevel("fatal")
	suite.Run(t, new(otpTestSuite))
}

func (s *otpTestSuite) SetupSuite() {
	s.httpClient = mockClient{}
	s.cfg = &env.OTPClient{Host: "http://otp:3000"}
	s.otpClient = otp.NewClient(s.cfg).WithHTTPClient(&s.httpClient)
}

func (s *otpTestSuite) TearDownTest() {
	s.httpClient.DoFunc = nil
}

func (s *otpTestSuite) TearDownSuite() {
}

func (s *otpTestSuite) TestGenerateByPhone() {
	ctx := reqContext()
	phone := "1"
	procID := "2"
	expires := time.Date(2000, 1, 1, 1, 1, 1, 1, time.UTC)
	expectedOTP := &entity.OTP{
		Code:      "1234",
		ExpiresAt: expires,
		Value:     phone,
	}
	s.httpClient.DoFunc = func(req *fasthttp.Request, resp *fasthttp.Response) error {
		s.assertReq(ctx, req, "generate", http.MethodPost)

		reqBody := make(map[string]string)
		err := json.Unmarshal(req.Body(), &reqBody)

		s.NoError(err)
		s.Equal(map[string]string{
			"type":      "PHONE",
			"value":     phone,
			"processID": procID,
		}, reqBody)

		resp.SetStatusCode(http.StatusOK)

		b, _ := json.Marshal(map[string]interface{}{"data": expectedOTP})
		_, _ = resp.BodyWriter().Write(b)

		return nil
	}

	r, err := s.otpClient.GenerateByPhone(ctx, phone, procID)

	s.NoError(err)
	s.Equal(expectedOTP, r)

	s.asserErrResp(func() error {
		_, err := s.otpClient.GenerateByPhone(ctx, phone, procID)
		return err
	})
}

func (s *otpTestSuite) TestValidate() {
	ctx := reqContext()
	phone := "1"
	procID := "2"
	p := &entity.ValidateOTPParams{
		Code:      "1234",
		Value:     phone,
		ProcessID: procID,
	}
	s.httpClient.DoFunc = func(req *fasthttp.Request, resp *fasthttp.Response) error {
		s.assertReq(ctx, req, "validate", http.MethodPost)

		reqBody := new(entity.ValidateOTPParams)
		err := json.Unmarshal(req.Body(), &reqBody)

		s.NoError(err)
		s.Equal(p, reqBody)

		resp.SetStatusCode(http.StatusOK)

		return nil
	}

	err := s.otpClient.Validate(ctx, p)

	s.NoError(err)

	s.asserErrResp(func() error {
		return s.otpClient.Validate(ctx, p)
	})
}

func (s *otpTestSuite) asserErrResp(callFn func() error) {
	expectedErr := cerror.NewF(context.Background(), cerror.KindConflict, "some conflict")
	s.httpClient.DoFunc = func(req *fasthttp.Request, resp *fasthttp.Response) error {
		resp.SetStatusCode(expectedErr.Kind().HTTPCode())

		b, _ := json.Marshal(cerror.BuildErrorResponse(expectedErr))
		_, _ = resp.BodyWriter().Write(b)

		return nil
	}

	err := callFn()
	s.Error(err)
	s.Equal(expectedErr.Kind().String(), cerror.ErrKind(err).String())
	s.Equal(fmt.Sprintf("otp service error: %s", expectedErr.Error()), err.Error())
}

func (s *otpTestSuite) assertReq(ctx context.Context, req *fasthttp.Request, expRoute, expMethod string) {
	s.Equal(fmt.Sprintf("%s/%s", s.cfg.Host, expRoute), req.URI().String())
	s.Equal(expMethod, string(req.Header.Method()))
	s.Equal("application/json", string(req.Header.ContentType()))

	for _, h := range consts.RequestHeadersToSave() {
		s.Equal(ctx.Value(h).(string), string(req.Header.Peek(h)))
	}
}

func reqContext() context.Context {
	ctx := context.Background()
	for _, h := range consts.RequestHeadersToSave() {
		ctx = context.WithValue(ctx, h, h) //nolint:staticcheck
	}

	return ctx
}
