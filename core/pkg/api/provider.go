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
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/token"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
)

// Provider is a dependency injection container containing essential utilities for API
// services.
type Provider struct {
	Config
	db       dbProvider
	user     userProvider
	access   accessProvider
	auth     authProvider
	cluster  clusterProvider
	ontology ontologyProvider
	status   statusProvider
}

func NewProvider(cfg Config) Provider {
	return Provider{
		Config: cfg,
		db:     dbProvider{DB: cfg.Distribution.DB},
		user:   userProvider{user: cfg.Service.User},
		access: accessProvider{access: cfg.Service.RBAC},
		auth: authProvider{
			authenticator: cfg.Service.Auth,
			token:         cfg.Service.Token,
		},
		cluster:  clusterProvider{cluster: cfg.Distribution.Cluster},
		ontology: ontologyProvider{ontology: cfg.Distribution.Ontology},
		status:   statusProvider{status: cfg.Service.Status},
	}
}

type dbProvider struct{ *gorp.DB }

type userProvider struct{ user *user.Service }

type accessProvider struct{ access *rbac.Service }

type authProvider struct {
	authenticator auth.Authenticator
	token         *token.Service
}

type ontologyProvider struct{ ontology *ontology.Ontology }

type clusterProvider struct{ cluster cluster.Cluster }

type statusProvider struct{ status *status.Service }
