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

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/constraint"
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

var (
	_             config.Config[ServiceConfig] = ServiceConfig{}
	DefaultConfig                              = ServiceConfig{}
)

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

func (s *Service) Close() error {
	c := errors.NewCatcher(errors.WithAggregation())
	c.Exec(s.Policy.Close)
	c.Exec(s.Role.Close)
	return c.Error()
}

func (s *Service) Enforce(ctx context.Context, req access.Request) error {
	return s.NewEnforcer(nil).Enforce(ctx, req)
}

func (s *Service) Filter(
	ctx context.Context,
	req access.Request,
) ([]ontology.ID, error) {
	return s.NewEnforcer(nil).Filter(ctx, req)
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

// OpenService creates a new RBAC service with both Policy and Role sub-services.
func OpenService(ctx context.Context, configs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}

	policyService, err := policy.OpenService(ctx, policy.Config{
		DB:       cfg.DB,
		Signals:  cfg.Signals,
		Ontology: cfg.Ontology,
	})
	if err != nil {
		return nil, err
	}

	roleService, err := role.OpenService(ctx, role.Config{
		DB:       cfg.DB,
		Ontology: cfg.Ontology,
		Signals:  cfg.Signals,
		Group:    cfg.Group,
	})
	if err != nil {
		return nil, err
	}
	s := &Service{
		Policy: policyService,
		Role:   roleService,
		cfg:    cfg,
	}
	return s, nil
}

func (e *Enforcer) retrievePolicies(
	ctx context.Context,
	subject ontology.ID,
) ([]policy.Policy, error) {
	var policies []policy.Policy
	if err := e.policy.NewRetrieve().
		WhereSubjects(subject).
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
// and policies from all roles assigned to the user. Returns ErrDenied if ANY object in
// the request is not accessible.
func (e *Enforcer) Enforce(ctx context.Context, req access.Request) error {
	policies, err := e.retrievePolicies(ctx, req.Subject)
	if err != nil {
		return err
	}

	for _, obj := range req.Objects {
		if !e.isObjectAccessible(ctx, policies, req, obj) {
			return access.ErrDenied
		}
	}

	return nil
}

// Filter implements the access.Enforcer interface. It returns only the objects from the
// request that the subject has access to. Unlike Enforce, this does not fail on denied
// objects - it simply excludes them from the result.
func (e *Enforcer) Filter(
	ctx context.Context,
	req access.Request,
) ([]ontology.ID, error) {
	policies, err := e.retrievePolicies(ctx, req.Subject)
	if err != nil {
		return nil, err
	}

	var accessible []ontology.ID
	for _, obj := range req.Objects {
		if e.isObjectAccessible(ctx, policies, req, obj) {
			accessible = append(accessible, obj)
		}
	}

	return accessible, nil
}

// isObjectAccessible checks if a single object is accessible given the policies.
func (e *Enforcer) isObjectAccessible(
	ctx context.Context,
	policies []policy.Policy,
	req access.Request,
	obj ontology.ID,
) bool {
	// Check DENY policies first - any match means access denied
	for _, p := range policies {
		if p.Effect != policy.EffectDeny {
			continue
		}
		if e.policyMatches(ctx, p, req, obj) {
			return false
		}
	}

	// Check ALLOW policies - need at least one match
	for _, p := range policies {
		if p.Effect == policy.EffectDeny {
			continue
		}
		if e.policyMatches(ctx, p, req, obj) {
			return true
		}
	}

	return false
}

// policyMatches checks if a policy matches the request for a specific object.
// It checks: action match, object match, and constraint satisfaction.
func (e *Enforcer) policyMatches(
	ctx context.Context,
	p policy.Policy,
	req access.Request,
	obj ontology.ID,
) bool {
	return e.actionMatches(p.Constraint, req.Action) &&
		e.objectMatches(p.Constraint, obj) &&
		e.constraintSatisfied(ctx, p.Constraint, req, obj)
}

// actionMatches checks if the constraint covers the requested action.
func (e *Enforcer) actionMatches(c constraint.Constraint, action access.Action) bool {
	// Empty actions list matches all (for system-wide policies)
	if len(c.Actions) == 0 {
		return true
	}
	return lo.Contains(c.Actions, action)
}

// objectMatches checks if the constraint covers the requested object.
func (e *Enforcer) objectMatches(c constraint.Constraint, obj ontology.ID) bool {
	// Empty objects list matches all (for system-wide policies)
	if len(c.Objects) == 0 {
		return true
	}
	for _, policyObj := range c.Objects {
		if policyObj.IsType() {
			if policyObj.Type == obj.Type {
				return true
			}
		} else if policyObj.Type == obj.Type && policyObj.Key == obj.Key {
			return true
		}
	}
	return false
}

// constraintSatisfied checks if the constraint is satisfied.
func (e *Enforcer) constraintSatisfied(
	ctx context.Context,
	c constraint.Constraint,
	req access.Request,
	obj ontology.ID,
) bool {
	// Build the enforce params with a single-element Objects slice
	singleObjReq := req
	singleObjReq.Objects = []ontology.ID{obj}
	params := constraint.EnforceParams{
		Request:  singleObjReq,
		Ontology: e.cfg.Ontology,
		Tx:       e.tx,
	}
	return c.Enforce(ctx, params)
}
