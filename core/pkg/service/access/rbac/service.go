// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/migrate"
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
}

var (
	_             config.Config[ServiceConfig] = ServiceConfig{}
	DefaultConfig                              = ServiceConfig{}
)

// Override implements [config.Config].
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Validate implements [config.Config].
func (c ServiceConfig) Validate() error {
	v := validate.New("rbac")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	return v.Error()
}

type Service struct {
	Policy *policy.Service
	Role   *role.Service
	cfg    ServiceConfig
}

func (s *Service) Close() error {
	c := errors.NewCatcher(errors.WithAggregation())
	c.Exec(s.Policy.Close)
	c.Exec(s.Role.Close)
	return c.Error()
}

func (s *Service) Enforce(ctx context.Context, req access.Request) error {
	return s.NewEnforcer(nil).Enforce(ctx, req)
}

// RetrievePoliciesForSubject retrieves all policies that apply to the given subject.
// This includes all policies from roles assigned to the subject via ontology relationships.
func (s *Service) RetrievePoliciesForSubject(
	ctx context.Context,
	subject ontology.ID,
	tx gorp.Tx,
) ([]policy.Policy, error) {
	return s.NewEnforcer(tx).retrievePolicies(ctx, subject)
}

// OpenService creates a new RBAC service with both Policy and Role sub-services.
func OpenService(ctx context.Context, configs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}

	policyService, err := policy.OpenService(ctx, policy.Config{
		DB:      cfg.DB,
		Signals: cfg.Signals,
	})
	if err != nil {
		return nil, err
	}

	roleService, err := role.OpenService(ctx, role.Config{
		DB:       cfg.DB,
		Ontology: cfg.Ontology,
		Signals:  cfg.Signals,
	})
	if err != nil {
		return nil, err
	}

	s := &Service{
		Policy: policyService,
		Role:   roleService,
		cfg:    cfg,
	}

	// Run policy migration from V0 (subject-based) to V1 (role-based)
	// This is idempotent and safe to run on every startup
	if err := migrate.MigratePolicies(ctx, cfg.DB, cfg.Ontology); err != nil {
		return nil, errors.Wrap(err, "failed to migrate policies to role-based model")
	}

	return s, nil
}

func (e *Enforcer) retrievePolicies(
	ctx context.Context,
	subject ontology.ID,
) ([]policy.Policy, error) {
	var (
		roleIDs  []ontology.Resource
		roles    []role.Role
		policies []policy.Policy
	)
	if err := e.cfg.Ontology.NewRetrieve().WhereIDs(subject).
		ExcludeFieldData(true).
		TraverseTo(role.Roles).Entries(&roleIDs).ExcludeFieldData(true).
		Exec(ctx, e.tx); err != nil {
		return nil, err
	}
	keys, err := role.KeysFromOntologyIds(ontology.ResourceIDs(roleIDs))
	if err != nil {
		return nil, err
	}
	if err := e.role.NewRetrieve().WhereKeys(keys...).
		Entries(&roles).
		Exec(ctx, e.tx); err != nil {
		return nil, err
	}
	policyKeys := lo.FlatMap(roles, func(item role.Role, index int) []uuid.UUID {
		return item.Policies
	})
	if err := e.policy.NewRetrieve().WhereKeys(policyKeys...).
		Entries(&policies).
		Exec(ctx, e.tx); err != nil {
		return nil, err
	}
	return policies, nil
}

type Enforcer struct {
	policy *policy.Service
	role   *role.Service
	cfg    ServiceConfig
	tx     gorp.Tx
}

func (s *Service) NewEnforcer(tx gorp.Tx) *Enforcer {
	return &Enforcer{
		role:   s.Role,
		policy: s.Policy,
		cfg:    s.cfg,
		tx:     gorp.OverrideTx(s.cfg.DB, tx),
	}
}

// Enforce implements the access.Enforcer interface. It checks both direct user policies
// and policies from all roles assigned to the user.
func (e *Enforcer) Enforce(ctx context.Context, req access.Request) error {
	v, err := e.retrievePolicies(ctx, req.Subject)
	if err != nil {
		return err
	}
	return lo.Ternary(allowRequest(req, v), access.Granted, access.Denied)
}

func allowRequest(req access.Request, policies []policy.Policy) bool {
	for _, requestedObj := range req.Objects {
		allowed := false
		for _, p := range policies {
			actionAllowed := lo.Contains(p.Actions, req.Action) || lo.Contains(p.Actions, access.ActionAll)
			if !actionAllowed {
				continue
			}
			for _, policyObj := range p.Objects {
				if policyObj.IsType() {
					if policyObj.Type == requestedObj.Type {
						allowed = true
						break
					}
				} else {
					if policyObj.Type == requestedObj.Type && policyObj.Key == requestedObj.Key {
						allowed = true
						break
					}
				}
			}
			if allowed {
				break
			}
		}
		if !allowed {
			return false
		}
	}
	return true
}
