// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api

import (
	"context"
	"github.com/go-playground/validator/v10"
	"github.com/synnaxlabs/synnax/pkg/access"
	errors "github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/synnax/pkg/auth"
	"github.com/synnaxlabs/synnax/pkg/auth/token"
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/user"
	"github.com/synnaxlabs/x/gorp"
)

// Provider is a dependency injection container containing essential utilities
// for particular API services (if they so require them).
type Provider struct {
	Config
	Validation validationProvider
	db         dbProvider
	user       userProvider
	access     AccessProvider
	auth       authProvider
	cluster    clusterProvider
	ontology   OntologyProvider
}

func NewProvider(cfg Config) Provider {
	p := Provider{Config: cfg}
	p.Validation = validationProvider{validator: newValidator()}
	p.db = dbProvider{db: gorp.Wrap(cfg.Storage.KV)}
	p.user = userProvider{user: cfg.User}
	p.access = AccessProvider{enforcer: cfg.Enforcer}
	p.auth = authProvider{token: cfg.Token, authenticator: cfg.Authenticator}
	p.cluster = clusterProvider{cluster: cfg.Cluster}
	p.ontology = OntologyProvider{Ontology: cfg.Ontology}
	return p
}

// validationProvider provides the global API validator to services.
type validationProvider struct {
	validator *validator.Validate
}

// Validate validates the provided struct. If Validation is successful, returns errors.Nil,
// otherwise, returns an errors.Validation error containing the fields that failed Validation.
func (vp *validationProvider) Validate(v any) errors.Typed {
	return errors.MaybeValidation(vp.validator.Struct(v))
}

// dbProvider provides exposes the cluster-wide key-value store to API services.
type dbProvider struct {
	db *gorp.DB
}

// WithWrite wraps the provided function in a gorp transaction. If the function returns
// errors.Nil, the transaction is committed. Otherwise, the transaction is aborted.
// Returns errors.Nil if the commit process is successful. Returns an unexpected
// error if the abort process fails; otherwise, returns the error returned by the provided
// function.
func (db dbProvider) WithWrite(ctx context.Context, f func(txn gorp.WriteContext) errors.Typed) (tErr errors.Typed) {
	txn := db.db.BeginWrite(ctx)
	defer func() {
		if err := txn.Close(); err != nil {
			tErr = errors.Unexpected(err)
		}
	}()
	tErr = f(txn)
	if !tErr.Occurred() {
		tErr = errors.MaybeUnexpected(txn.Commit(ctx))
	}
	return tErr
}

// userProvider provides user information to services.
type userProvider struct {
	user *user.Service
}

// AccessProvider provides access control information and utilities to services.
type AccessProvider struct {
	enforcer access.Enforcer
}

// authProvider provides authentication and token utilities to services. In most cases
// authentication should be left up to the protocol-specific middleware.
type authProvider struct {
	authenticator auth.Authenticator
	token         *token.Service
}

// OntologyProvider provides the cluster wide ontology to services.
type OntologyProvider struct {
	Ontology *ontology.Ontology
}

// clusterProvider provides cluster topology information to services.
type clusterProvider struct {
	cluster dcore.Cluster
}
