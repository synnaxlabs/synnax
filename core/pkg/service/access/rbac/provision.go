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

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

// provisionResult contains the keys of all provisioned built-in roles.
type provisionResult struct {
	ownerKey    uuid.UUID
	engineerKey uuid.UUID
	operatorKey uuid.UUID
	viewerKey   uuid.UUID
}

func (s *Service) provision(ctx context.Context, tx gorp.Tx) (provisionResult, error) {
	var result provisionResult
	var err error
	if result.viewerKey, err = s.provisionRole(ctx, viewerRole, []policy.Policy{viewerPolicy}, tx); err != nil {
		return provisionResult{}, err
	}
	if result.operatorKey, err = s.provisionRole(ctx, operatorRole, operatorPolicies, tx); err != nil {
		return provisionResult{}, err
	}
	if result.engineerKey, err = s.provisionRole(ctx, engineerRole, engineerPolicies, tx); err != nil {
		return provisionResult{}, err
	}
	if result.ownerKey, err = s.provisionRole(ctx, ownerRole, []policy.Policy{ownerPolicy}, tx); err != nil {
		return provisionResult{}, err
	}
	return result, nil
}

func (s *Service) provisionRole(
	ctx context.Context,
	rol role.Role,
	policies []policy.Policy,
	tx gorp.Tx,
) (uuid.UUID, error) {
	policyKeys := make([]uuid.UUID, 0, len(policies))
	for i := range policies {
		pol := &policies[i]
		desiredObjects := pol.Objects
		desiredActions := pol.Actions
		if err := s.Policy.NewRetrieve().
			WhereNames(pol.Name).
			Entry(pol).
			Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
			return uuid.Nil, err
		}
		pol.Objects = desiredObjects
		pol.Actions = desiredActions
		if err := s.Policy.NewWriter(tx, true).Create(ctx, pol); err != nil {
			return uuid.Nil, err
		}
		policyKeys = append(policyKeys, pol.Key)
	}
	if err := s.Role.NewRetrieve().
		WhereName(rol.Name).
		Entry(&rol).
		Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
		return uuid.Nil, err
	}
	if rol.Key == uuid.Nil {
		w := s.Role.NewWriter(tx, true)
		if err := w.Create(ctx, &rol); err != nil {
			return uuid.Nil, err
		}
		if err := s.Policy.NewWriter(tx, true).SetOnRole(ctx, rol.Key, policyKeys...); err != nil {
			return uuid.Nil, err
		}
	}
	return rol.Key, nil
}
