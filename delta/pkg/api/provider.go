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

// Provider is a dependency injection container containing essential utilities
// for particular API services (if they so require them). Provider should be
// passed as the one and only argument to a service constructor.
type Provider struct {
	Config     Config
	Logging    LoggingProvider
	Validation ValidationProvider
	DB         DBProvider
	User       UserProvider
	Access     AccessProvider
	Auth       AuthProvider
}

func NewProvider(cfg Config) Provider {
	p := Provider{Config: cfg}
	p.Logging = LoggingProvider{Logger: cfg.Logger.Sugar()}
	p.Validation = ValidationProvider{Validator: newValidator()}
	p.DB = DBProvider{DB: gorp.Wrap(cfg.Storage.KV)}
	p.User = UserProvider{User: cfg.User}
	p.Access = AccessProvider{Enforcer: cfg.Enforcer}
	p.Auth = AuthProvider{Token: cfg.Token, Authenticator: cfg.Authenticator}
	return p
}

// LoggingProvider provides logging utilities to services.
type LoggingProvider struct {
	Logger *zap.SugaredLogger
}

// ValidationProvider provides the global API validator to services.
type ValidationProvider struct {
	Validator *validator.Validate
}

// Validate validates the provided struct. If validation is successful, returns errors.Nil,
// otherwise, returns an errors.Validation error containing the fields that failed validation.
func (vp *ValidationProvider) Validate(v any) errors.Typed {
	return errors.MaybeValidation(vp.Validator.Struct(v))
}

// DBProvider provides exposes the cluster-wide key-value store to API services.
type DBProvider struct {
	DB *gorp.DB
}

// WithTxn wraps the provided function in a gorp transaction. If the function returns
// errors.Nil, the transaction is committed. Otherwise, the transaction is aborted.
// Returns errors.Nil if the commit process is successful. Returns an unexpected
// error if the abort process fails; otherwise, returns the error returned by the provided
// function.
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

// UserProvider provides user information to services.
type UserProvider struct {
	User *user.Service
}

// AccessProvider provides access control information and utilities to services.
type AccessProvider struct {
	Enforcer access.Enforcer
}

// AuthProvider provides authentication and token utilities to services. In most cases
// authentication should be left up to the protocol-specific middleware.
type AuthProvider struct {
	Authenticator auth.Authenticator
	Token         *token.Service
}

// OntologyProvider provides the cluster wide ontology to services.
type OntologyProvider struct {
	Ontology *ontology.Ontology
}
