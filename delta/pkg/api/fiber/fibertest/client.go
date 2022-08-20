package fibertest

import (
	"bytes"
	"github.com/arya-analytics/x/httputil"
	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"
	"net/http"
	"net/http/httptest"
)

type Client struct {
	EncoderDecoder httputil.EncoderDecoder
	App            *fiber.App
	Headers        map[string]string
}

func DefaultClient() *Client {
	return &Client{
		EncoderDecoder: &httputil.JSONEncoderDecoder{},
		App:            fiber.New(),
		Headers:        make(map[string]string),
	}
}

func (c *Client) Post(
	target string,
	body interface{},
) (*http.Response, error) {
	req := httptest.NewRequest(
		http.MethodPost,
		target,
		bytes.NewReader(lo.Must(c.EncoderDecoder.Encode(body))),
	)
	req.Header.Set("Content-Type", c.EncoderDecoder.ContentType())
	c.applyHeaders(req)
	return c.App.Test(req)
}

func (c *Client) Decode(
	res *http.Response,
	out interface{},
) error {
	// read the body into a buffer
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(res.Body)
	if err != nil {
		return err
	}
	return c.EncoderDecoder.Decode(buf.Bytes(), out)
}

func (c *Client) Get(target string) *http.Request {
	req := httptest.NewRequest(
		http.MethodGet,
		target,
		nil,
	)
	c.applyHeaders(req)
	return req
}

func (c *Client) SetHeader(key, value string) {
	c.Headers[key] = value
}

func (c *Client) applyHeaders(req *http.Request) {
	for key, value := range c.Headers {
		req.Header.Set(key, value)
	}
}
