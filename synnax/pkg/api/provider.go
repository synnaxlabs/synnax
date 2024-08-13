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
	"github.com/synnaxlabs/synnax/pkg/access"
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
	db       dbProvider
	user     userProvider
	access   accessProvider
	auth     authProvider
	cluster  clusterProvider
	ontology OntologyProvider
}

func NewProvider(cfg Config) Provider {
	p := Provider{Config: cfg}
	p.db = dbProvider{DB: gorp.Wrap(cfg.Storage.KV)}
	p.user = userProvider{user: cfg.User}
	p.access = accessProvider{access: cfg.Enforcer} // originally RBAC
	p.auth = authProvider{token: cfg.Token, authenticator: cfg.Authenticator}
	p.cluster = clusterProvider{cluster: cfg.Cluster}
	p.ontology = OntologyProvider{Ontology: cfg.Ontology}
	return p
}

// dbProvider provides exposes the cluster-wide key-value store to API services.
type dbProvider struct {
	*gorp.DB
}

// userProvider provides user information to services.
type userProvider struct {
	user *user.Service
}

// AccessProvider provides access control information and utilities to services.
type accessProvider struct {
	access access.Enforcer
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
