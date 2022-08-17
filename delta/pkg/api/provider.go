package api

import (
	"github.com/arya-analytics/delta/pkg/access"
	errors "github.com/arya-analytics/delta/pkg/api/errors"
	"github.com/arya-analytics/delta/pkg/auth"
	"github.com/arya-analytics/delta/pkg/auth/token"
	"github.com/arya-analytics/delta/pkg/distribution/ontology"
	"github.com/arya-analytics/delta/pkg/user"
	"github.com/arya-analytics/x/gorp"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type Provider struct {
	Config Config
	Core   CoreProvider
	DB     DBProvider
	User   UserProvider
	Access AccessProvider
	Auth   AuthProvider
}

func NewProvider(cfg Config) Provider {
	p := Provider{Config: cfg}

	p.Core = CoreProvider{Logger: cfg.Logger.Sugar(), Validator: newValidator()}

	p.DB = DBProvider{Core: p.Core, DB: gorp.Wrap(cfg.Storage.KV)}

	p.User = UserProvider{User: cfg.User}

	p.Access = AccessProvider{Enforcer: cfg.Enforcer}

	p.Auth = AuthProvider{Token: cfg.Token, Authenticator: cfg.Authenticator}

	return p
}

type CoreProvider struct {
	Logger    *zap.SugaredLogger
	Validator *validator.Validate
}

func (c *CoreProvider) Validate(v any) errors.Typed {
	return errors.MaybeValidation(c.Validator.Struct(v))
}

type DBProvider struct {
	DB   *gorp.DB
	Core CoreProvider
}

func (db DBProvider) WithTxn(f func(txn gorp.Txn) errors.Typed) (tErr errors.Typed) {
	txn := db.DB.BeginTxn()
	defer func() {
		if err := txn.Close(); err != nil {
			tErr = errors.Unexpected(err)
		}
	}()
	tErr = f(txn)
	if !tErr.Occurred() {
		tErr = errors.MaybeUnexpected(txn.Commit())
	}
	return tErr
}

type UserProvider struct {
	User *user.Service
}

type AccessProvider struct {
	Enforcer access.Enforcer
}

type AuthProvider struct {
	Authenticator auth.Authenticator
	Token         *token.Service
}

type Map map[string]interface{}

type OntologyProvider struct {
	Ontology *ontology.Ontology
}
