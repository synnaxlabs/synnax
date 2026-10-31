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
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// ProvisionResult contains the keys of all provisioned built-in roles.
type ProvisionResult struct {
	// OwnerKey is the key of the Owner role.
	OwnerKey role.Key
	// EngineerKey is the key of the Engineer role.
	EngineerKey role.Key
	// OperatorKey is the key of the Operator role.
	OperatorKey role.Key
	// DriverKey is the key of the Driver role, which every rack holds.
	DriverKey role.Key
	// ViewerKey is the key of the Viewer role.
	ViewerKey role.Key
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
		if result.ViewerKey, err = provisionRole(
			ctx,
			viewerRole,
			[]policy.Policy{viewerPolicy},
			tx,
			policySvc,
			roleSvc,
		); err != nil {
			return err
		}
		if result.DriverKey, err = provisionRole(
			ctx,
			driverRole,
			driverPolicies,
			tx,
			policySvc,
			roleSvc,
		); err != nil {
			return err
		}
		if result.OperatorKey, err = provisionRole(
			ctx,
			operatorRole,
			operatorPolicies,
			tx,
			policySvc,
			roleSvc,
		); err != nil {
			return err
		}
		if result.EngineerKey, err = provisionRole(
			ctx,
			engineerRole,
			engineerPolicies,
			tx,
			policySvc,
			roleSvc,
		); err != nil {
			return err
		}
		if result.OwnerKey, err = provisionRole(
			ctx,
			ownerRole,
			[]policy.Policy{ownerPolicy},
			tx,
			policySvc,
			roleSvc,
		); err != nil {
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
) (role.Key, error) {
	policyKeys := make([]policy.Key, 0, len(policies))
	for i := range policies {
		pol := &policies[i]
		desiredName := pol.Name
		desiredObjects := pol.Objects
		desiredActions := pol.Actions
		if err := policySvc.NewRetrieve().
			Where(policy.MatchNames(storedNames(pol.Name)...)).
			Where(policy.MatchInternal(true)).
			Entry(pol).
			Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
			return uuid.Nil(), err
		}
		pol.Name = desiredName
		pol.Objects = desiredObjects
		pol.Actions = desiredActions
		if err := policySvc.NewWriter(tx, true).Create(ctx, pol); err != nil {
			return uuid.Nil(), err
		}
		policyKeys = append(policyKeys, pol.Key)
	}
	desired := rol
	if err := roleSvc.NewRetrieve().
		Where(role.MatchNames(storedNames(rol.Name)...)).
		Where(role.MatchInternal(true)).
		Entry(&rol).
		Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
		return uuid.Nil(), err
	}
	if rol.Key == uuid.Nil() || rol.Name != desired.Name {
		rol.Name = desired.Name
		rol.Description = desired.Description
		if err := roleSvc.NewWriter(tx, true).Create(ctx, &rol); err != nil {
			return uuid.Nil(), err
		}
	}
	if err := policySvc.NewWriter(tx, true).
		SetOnRole(ctx, rol.Key, policyKeys...); err != nil {
		return uuid.Nil(), err
	}
	return rol.Key, nil
}

func storedNames(name string) []string {
	if former, ok := formerNames[name]; ok {
		return []string{name, former}
	}
	return []string{name}
}
