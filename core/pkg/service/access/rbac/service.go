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

// ServiceConfig is the configuration for opening the RBAC service.
type ServiceConfig struct {
	DB       *gorp.DB
	Ontology *ontology.Ontology
	Signals  *signals.Provider
	Group    *group.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Group = override.Nil(c.Group, other.Group)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("rbac")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "group", c.Group)
	return v.Error()
}

// Service is the main entrypoint for the RBAC service. It provides methods for
// retrieving policies for a subject and enforcing access policies.
type Service struct {
	// Policy is the service for managing policies.
	Policy *policy.Service
	// Role is the service for managing roles.
	Role *role.Service
	cfg  ServiceConfig
	// defaultPolicies is a list of policies that are applied to all subjects.
	defaultPolicies []policy.Policy
}

// OpenService creates a new RBAC service with both Policy and Role sub-services.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
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
		return nil, errors.Combine(err, s.Policy.Close())
	}
	return s, nil
}

// AddDefaultPolicy adds a default policy to the service. This policy will be retrieved
// and applied to all subjects.
func (s *Service) AddDefaultPolicy(policy policy.Policy) {
	s.defaultPolicies = append(s.defaultPolicies, policy)
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

// Close closes the RBAC service and releases any resources that it may have acquired.
func (s *Service) Close() error {
	c := errors.NewCatcher(errors.WithAggregation())
	c.Exec(s.Policy.Close)
	c.Exec(s.Role.Close)
	return c.Error()
}
