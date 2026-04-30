// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package builtin

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

// ProvisionResult contains the keys of all provisioned built-in roles.
type ProvisionResult struct {
	OwnerKey    uuid.UUID
	EngineerKey uuid.UUID
	OperatorKey uuid.UUID
	ViewerKey   uuid.UUID
}

// Provision creates or updates all built-in roles and their associated policies.
// This is idempotent and should be called on every startup to ensure policy
// definitions stay up to date.
func Provision(
	ctx context.Context,
	db *gorp.DB,
	policySvc *policy.Service,
	roleSvc *role.Service,
) (result ProvisionResult, err error) {
	err = db.WithTx(ctx, func(tx gorp.Tx) error {
		if result.ViewerKey, err = provisionRole(ctx, viewerRole, []policy.Policy{viewerPolicy}, tx, policySvc, roleSvc); err != nil {
			return err
		}
		if result.OperatorKey, err = provisionRole(ctx, operatorRole, operatorPolicies, tx, policySvc, roleSvc); err != nil {
			return err
		}
		if result.EngineerKey, err = provisionRole(ctx, engineerRole, engineerPolicies, tx, policySvc, roleSvc); err != nil {
			return err
		}
		if result.OwnerKey, err = provisionRole(ctx, ownerRole, []policy.Policy{ownerPolicy}, tx, policySvc, roleSvc); err != nil {
			return err
		}
		return nil
	})
	return result, err
}

func provisionRole(
	ctx context.Context,
	rol role.Role,
	policies []policy.Policy,
	tx gorp.Tx,
	policySvc *policy.Service,
	roleSvc *role.Service,
) (uuid.UUID, error) {
	policyKeys := make([]uuid.UUID, 0, len(policies))
	for i := range policies {
		pol := &policies[i]
		desiredObjects := pol.Objects
		desiredActions := pol.Actions
		if err := policySvc.NewRetrieve().
			Where(policy.MatchNames(pol.Name)).
			Entry(pol).
			Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
			return uuid.Nil, err
		}
		pol.Objects = desiredObjects
		pol.Actions = desiredActions
		if err := policySvc.NewWriter(tx, true).Create(ctx, pol); err != nil {
			return uuid.Nil, err
		}
		policyKeys = append(policyKeys, pol.Key)
	}
	if err := roleSvc.NewRetrieve().
		Where(role.MatchNames(rol.Name)).
		Entry(&rol).
		Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
		return uuid.Nil, err
	}
	if rol.Key == uuid.Nil {
		w := roleSvc.NewWriter(tx, true)
		if err := w.Create(ctx, &rol); err != nil {
			return uuid.Nil, err
		}
	}
	if err := policySvc.NewWriter(tx, true).SetOnRole(ctx, rol.Key, policyKeys...); err != nil {
		return uuid.Nil, err
	}
	return rol.Key, nil
}
