package fws

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"net/http"
)

// ClientTokenAuth implements simple token authorization for the Client. To initialize,
// simply pass the token as a string.
type ClientTokenAuth string

const AuthorizationValuePrefix = "Bearer "

func (s ClientTokenAuth) Handle(
	ctx context.Context,
	req DialRequest,
	next func() (*http.Response, error),
) (*http.Response, error) {
	req.Header.Set(fiber.HeaderAuthorization, AuthorizationValuePrefix+string(s))
	return next()
}
