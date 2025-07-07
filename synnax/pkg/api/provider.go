// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/token"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
)

// Provider is a dependency injection container containing essential utilities for
// particular API services (if they so require them).
type Provider struct {
	Config
	db       dbProvider
	user     userProvider
	access   accessProvider
	auth     authProvider
	framer   framerProvider
	channel  channelProvider
	cluster  clusterProvider
	ontology ontologyProvider
}

func NewProvider(cfg Config) Provider {
	p := Provider{Config: cfg}
	p.db = dbProvider{DB: cfg.Distribution.DB}
	p.user = userProvider{user: cfg.Service.User}
	p.access = accessProvider{access: cfg.Service.RBAC}
	p.auth = authProvider{token: cfg.Service.Token, authenticator: cfg.Service.Auth}
	p.cluster = clusterProvider{cluster: cfg.Distribution.Cluster}
	p.framer = framerProvider{framer: cfg.Service.Framer}
	p.channel = channelProvider{channel: &cfg.Distribution.Channel}
	p.ontology = ontologyProvider{Ontology: cfg.Distribution.Ontology}
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

// accessProvider provides access control information and utilities to services.
type accessProvider struct {
	access *rbac.Service
}

// framerProvider provides framer information to services.
type framerProvider struct {
	framer *framer.Service
}

type channelProvider struct {
	channel *channel.Service
}

// authProvider provides authentication and token utilities to services. In most cases
// authentication should be left up to the protocol-specific middleware.
type authProvider struct {
	authenticator auth.Authenticator
	token         *token.Service
}

// ontologyProvider provides the cluster wide ontology to services.
type ontologyProvider struct {
	Ontology *ontology.Ontology
}

// clusterProvider provides cluster topology information to services.
type clusterProvider struct {
	cluster cluster.Cluster
}
