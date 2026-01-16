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
// the request is not accessible. If a policy has an EffectDeny, it will return
// access.ErrDenied.
func (e *Enforcer) Enforce(ctx context.Context, req access.Request) error {
	policies, err := e.retrievePolicies(ctx, req.Subject)
	if err != nil {
		return err
	}
	for _, p := range policies {
		if p.Effect == policy.EffectDeny {
			matches, err := p.Constraint.Enforce(ctx, constraint.EnforceParams{
				Request:  req,
				Ontology: e.cfg.Ontology,
				Tx:       e.tx,
			})
			if err != nil {
				return err
			}
			if matches {
				return access.ErrDenied
			}
		}
	}
	for _, p := range policies {
		if p.Effect == policy.EffectAllow {
			if matches, err := p.Constraint.Enforce(ctx, constraint.EnforceParams{
				Request:  req,
				Ontology: e.cfg.Ontology,
				Tx:       e.tx,
			}); err != nil || matches {
				return err
			}
		}
	}
	return access.ErrDenied
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
