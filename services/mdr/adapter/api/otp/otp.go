package otp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"

	"wasfaty.api/pkg/cerror"
	"wasfaty.api/pkg/env"
	"wasfaty.api/pkg/http/headers"
	"wasfaty.api/services/mpi/entity"
)

const (
	typePhone = "PHONE"
)

type generateReq struct {
	Type      string `json:"type"`
	Value     string `json:"value"`
	ProcessID string `json:"processID"`
}

type generateResp struct {
	Data entity.OTP `json:"data"`
}

type HTTPClient interface {
	Do(req *fasthttp.Request, resp *fasthttp.Response) error
	DoTimeout(req *fasthttp.Request, resp *fasthttp.Response, timeout time.Duration) error
}

type Client struct {
	cfg        *env.OTPClient
	httpClient HTTPClient
}

func NewClient(cfg *env.OTPClient) *Client {
	return &Client{cfg: cfg, httpClient: new(fasthttp.Client)}
}

func (c *Client) WithHTTPClient(h HTTPClient) *Client {
	c.httpClient = h
	return c
}

func (c *Client) GenerateByPhone(ctx context.Context, phone, processID string) (*entity.OTP, error) {
	b, err := json.Marshal(generateReq{
		Type:      typePhone,
		Value:     phone,
		ProcessID: processID,
	})

	if err != nil {
		return nil, cerror.New(ctx, cerror.KindInternal, err).LogError()
	}

	dst := new(generateResp)

	if err := c.sendRequest(ctx, c.cfg.Host+"/generate", fiber.MethodPost, nil, nil, b, dst); err != nil {
		return nil, err
	}

	return &dst.Data, nil
}

func (c *Client) Validate(ctx context.Context, p *entity.ValidateOTPParams) error {
	b, err := json.Marshal(p)

	if err != nil {
		return cerror.New(ctx, cerror.KindInternal, err).LogError()
	}

	if err := c.sendRequest(ctx, c.cfg.Host+"/validate", fiber.MethodPost, nil, nil, b, nil); err != nil {
		return err
	}

	return nil
}

func (c *Client) sendRequest(
	ctx context.Context,
	url, method string,
	qParams, hdrs map[string]string,
	body []byte,
	dst interface{}) error {
	var err error

	req := fasthttp.AcquireRequest()

	defer fasthttp.ReleaseRequest(req)
	req.Header.SetContentType(fiber.MIMEApplicationJSON)
	req.Header.SetMethod(method)

	headers.AddHeadersFromContext(ctx, req)

	for k, v := range hdrs {
		req.Header.Set(k, v)
	}

	req.SetRequestURI(url)

	for k, v := range qParams {
		req.URI().QueryArgs().Set(k, v)
	}

	req.SetBody(body)

	resp := fasthttp.AcquireResponse()

	defer fasthttp.ReleaseResponse(resp)

	if c.cfg.RequestTimeout != 0 {
		err = c.httpClient.DoTimeout(req, resp, c.cfg.RequestTimeout)
	} else {
		err = c.httpClient.Do(req, resp)
	}

	if err != nil {
		return cerror.New(ctx, cerror.KindInternal, err).LogError()
	}

	respStatus := resp.StatusCode()
	respBody := resp.Body()

	if !(respStatus >= 200 && respStatus < 300) {
		var payload cerror.ResponseErrorWrap

		var errMsg string
		if err := json.Unmarshal(respBody, &payload); err == nil && payload.Error.Message != "" {
			errMsg = fmt.Sprintf("otp service error: %s", payload.Error.Message)
		} else {
			errMsg = fmt.Sprintf("otp service error. Code: %d", respStatus)
		}

		cErr := cerror.NewF(ctx, cerror.KindFromHTTPCode(respStatus), errMsg).WithPayload(payload).LogError()

		return cErr.WithPayload(nil)
	}

	if dst != nil {
		if err := json.Unmarshal(respBody, dst); err != nil {
			return cerror.New(ctx, cerror.KindInternal, err).LogError()
		}
	}

	return nil
}
