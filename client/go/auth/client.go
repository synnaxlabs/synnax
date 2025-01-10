package auth

import (
	"context"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
)

type (
	LoginRequest        = api.AuthLoginRequest
	LoginResponse       = api.AuthLoginResponse
	LoginClient         = freighter.UnaryClient[LoginRequest, LoginResponse]
	InsecureCredentials = auth.InsecureCredentials
)

type Client struct {
	transport     LoginClient
	creds         auth.InsecureCredentials
	authenticated bool
	token         string
}

const authorizationHeader = "Authorization"

func New(transport LoginClient, creds auth.InsecureCredentials) *Client {
	return &Client{transport: transport, creds: creds}
}

func (c *Client) authenticate(ctx context.Context) (string, error) {
	if c.authenticated {
		return c.token, nil
	}
	res, err := c.transport.Send(ctx, "", LoginRequest{
		InsecureCredentials: c.creds,
	})
	if err != nil {
		return "", err
	}
	c.token = res.Token
	c.authenticated = true
	return c.token, nil
}

func (c *Client) Middleware() freighter.Middleware {
	return freighter.MiddlewareFunc(func(ctx freighter.Context, next freighter.Next) (freighter.Context, error) {
		tk, err := c.authenticate(ctx)
		if err != nil {
			return ctx, err
		}
		ctx.Set(authorizationHeader, "Bearer "+tk)
		return next(ctx)
	})
}
