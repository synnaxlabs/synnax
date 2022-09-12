package fiber

import (
	apierrors "github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/synnax/pkg/auth/token"
	"github.com/synnaxlabs/synnax/pkg/user"
	"github.com/cockroachdb/errors"
	"github.com/gofiber/fiber/v2"
	"strings"
)

func tokenMiddleware(svc *token.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tk, err := parseToken(c)
		if err.Occurred() {
			return errorResponse(c, err)
		}
		key, vErr := svc.Validate(tk)
		if vErr != nil {
			return errorResponse(c, apierrors.General(err))
		}
		setSubject(c, user.OntologyID(key))
		return c.Next()
	}
}

type tokenParser func(c *fiber.Ctx) (token string, found bool, err apierrors.Typed)

const (
	tokenCookieName               = "Token"
	headerTokenPrefix             = "Bearer "
	invalidAuthorizationHeaderMsg = `
	invalid authorization header. Format should be

		'Authorization: Bearer <Token>'
	`
)

var tokenParsers = []tokenParser{
	typeParseCookieToken,
	tryParseHeaderToken,
}

func parseToken(c *fiber.Ctx) (string, apierrors.Typed) {
	for _, tp := range tokenParsers {
		if tk, found, err := tp(c); found {
			return tk, err
		}
	}
	return "", apierrors.Auth(errors.New(
		"request unauthorized - unable to parse token"))
}

func typeParseCookieToken(c *fiber.Ctx) (string, bool, apierrors.Typed) {
	tk := c.Cookies(tokenCookieName)
	return tk, len(tk) != 0, apierrors.Nil
}

const headerKey = "Authorization"

func tryParseHeaderToken(c *fiber.Ctx) (string, bool, apierrors.Typed) {
	authHeader := c.Get(headerKey)
	if len(authHeader) == 0 {
		return "", false, apierrors.Nil
	}
	splitToken := strings.Split(authHeader, headerTokenPrefix)
	if len(splitToken) != 2 {
		return "",
			false,
			apierrors.Auth(errors.New(invalidAuthorizationHeaderMsg))
	}
	return splitToken[1], true, apierrors.Nil
}
