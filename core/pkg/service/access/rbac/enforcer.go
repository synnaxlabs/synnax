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

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/constraint"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/set"
)

// Enforcer is an implementation of the access.Enforcer interface for the RBAC model.
type Enforcer struct {
	policy *policy.Service
	cfg    ServiceConfig
	tx     gorp.Tx
}

var _ access.Enforcer = &Enforcer{}

// NewEnforcer creates a new Enforcer with the given transaction.
func (s *Service) NewEnforcer(tx gorp.Tx) *Enforcer {
	return &Enforcer{
		policy: s.Policy,
		cfg:    s.cfg,
		tx:     gorp.OverrideTx(s.cfg.DB, tx),
	}
}

// Enforce implements the access.Enforcer interface. It checks both direct user policies
// and policies from all roles assigned to the user. Returns ErrDenied if ANY object in
// the request is explicitly denied OR if any object is not covered by an allow policy.
func (e *Enforcer) Enforce(ctx context.Context, req access.Request) error {
	policies, err := e.retrievePolicies(ctx, req.Subject)
	if err != nil {
		return err
	}

	params := constraint.EnforceParams{
		Request:  req,
		Ontology: e.cfg.Ontology,
		Tx:       e.tx,
	}

	// First, check deny policies. If any deny policy covers any requested object,
	// deny the entire request.
	for _, p := range policies {
		if p.Effect == policy.EffectDeny {
			deniedObjects, err := p.Constraint.Enforce(ctx, params)
			if err != nil {
				return err
			}
			if len(deniedObjects) > 0 {
				return access.ErrDenied
			}
		}
	}

	// Track which objects are covered by allow policies.
	coveredSet := make(set.Set[ontology.ID])
	for _, p := range policies {
		if p.Effect == policy.EffectAllow {
			allowedObjects, err := p.Constraint.Enforce(ctx, params)
			if err != nil {
				return err
			}
			for _, obj := range allowedObjects {
				coveredSet.Add(obj)
			}
		}
	}

	// Check if all requested objects are covered by at least one allow policy.
	for _, obj := range req.Objects {
		if !coveredSet.Contains(obj) {
			return access.ErrDenied
		}
	}

	return nil
}

func (e *Enforcer) retrievePolicies(
	ctx context.Context,
	subject ontology.ID,
) ([]policy.Policy, error) {
	var policies []policy.Policy
	if err := e.policy.NewRetrieve().
		WhereSubject(subject).
		Entries(&policies).
		Exec(ctx, e.tx); err != nil {
		return nil, err
	}
	return append(policies, e.policy.SystemPolicies()...), nil
}
