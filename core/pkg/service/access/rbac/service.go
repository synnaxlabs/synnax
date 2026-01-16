// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rbac

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type ServiceConfig struct {
	DB       *gorp.DB
	Ontology *ontology.Ontology
	Signals  *signals.Provider
	Group    *group.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements [config.Config].
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Group = override.Nil(c.Group, other.Group)
	return c
}

// Validate implements [config.Config].
func (c ServiceConfig) Validate() error {
	v := validate.New("rbac")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "group", c.Group)
	return v.Error()
}

type Service struct {
	Policy *policy.Service
	Role   *role.Service
	cfg    ServiceConfig
}

// OpenService creates a new RBAC service with both Policy and Role sub-services.
func OpenService(ctx context.Context, configs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, configs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	if s.Policy, err = policy.OpenService(ctx, policy.ServiceConfig{
		DB:       cfg.DB,
		Signals:  cfg.Signals,
		Ontology: cfg.Ontology,
	}); err != nil {
		return nil, err
	}
	if s.Role, err = role.OpenService(ctx, role.ServiceConfig{
		DB:       cfg.DB,
		Ontology: cfg.Ontology,
		Signals:  cfg.Signals,
		Group:    cfg.Group,
	}); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Service) Close() error {
	c := errors.NewCatcher(errors.WithAggregation())
	c.Exec(s.Policy.Close)
	c.Exec(s.Role.Close)
	return c.Error()
}

// RetrievePoliciesForSubject retrieves all policies that apply to the given subject.
// This includes all policies from roles assigned to the subject via ontology
// relationships.
func (s *Service) RetrievePoliciesForSubject(
	ctx context.Context,
	subject ontology.ID,
	tx gorp.Tx,
) ([]policy.Policy, error) {
	return s.NewEnforcer(tx).retrievePolicies(ctx, subject)
}
