package api

import (
	"context"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/freighter"
	apierrors "github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/synnax/pkg/auth/token"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/user"
)

func tokenMiddleware(svc *token.Service) freighter.Middleware {
	return freighter.MiddlewareFunc(func(
		ctx context.Context,
		md freighter.MD,
		next freighter.Next,
	) error {
		tk, _err := tryParseToken(md.Params)
		if _err.Occurred() {
			return _err
		}
		userKey, err := svc.Validate(tk)
		if err != nil {
			return apierrors.Auth(err)
		}
		setSubject(md.Params, user.OntologyID(userKey))
		return next(ctx, md)
	})
}

const tokenParamPrefix = "Bearer "

var (
	invalidAuthenticationParam = apierrors.Auth(errors.New(
		`invalid authorization token. Format should be
		'Authorization: Bearer <token>'`,
	))
	noAuthenticationParam = apierrors.Auth(errors.New("no authentication token provided"))
)

func tryParseToken(p freighter.Params) (string, apierrors.Typed) {
	tkParam, ok := p.Get(fiber.HeaderAuthorization)
	if !ok {
		return "", noAuthenticationParam
	}
	tkStr, ok := tkParam.(string)
	if !ok {
		return "", noAuthenticationParam
	}
	if !strings.HasPrefix(tkStr, tokenParamPrefix) {
		return "", invalidAuthenticationParam
	}
	tkStr = strings.TrimPrefix(tkStr, tokenParamPrefix)
	if !ok {
		return "", apierrors.Auth(errors.New("token not found"))
	}
	return tkStr, apierrors.Nil
}

const subjectKey = "Subject"

func setSubject(p freighter.Params, subject ontology.ID) {
	p.Set(subjectKey, subject)
}

func getSubject(ctx context.Context) ontology.ID {
	s, ok := freighter.MDFromContext(ctx).Params.Get(subjectKey)
	if !ok {
		panic("[api] - no subject found in context")
	}
	return s.(ontology.ID)
}
