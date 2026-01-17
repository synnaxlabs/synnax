// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package access

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
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

func Provision(
	ctx context.Context,
	tx gorp.Tx,
	service *rbac.Service,
) (ProvisionResult, error) {
	var result ProvisionResult
	var err error

	if result.ViewerKey, err = provisionRole(ctx, viewerRole, []policy.Policy{viewerPolicy}, tx, service); err != nil {
		return ProvisionResult{}, err
	}
	if result.OperatorKey, err = provisionRole(ctx, operatorRole, operatorPolicies, tx, service); err != nil {
		return ProvisionResult{}, err
	}
	if result.EngineerKey, err = provisionRole(ctx, engineerRole, engineerPolicies, tx, service); err != nil {
		return ProvisionResult{}, err
	}
	if result.OwnerKey, err = provisionRole(ctx, ownerRole, []policy.Policy{ownerPolicy}, tx, service); err != nil {
		return ProvisionResult{}, err
	}
	return result, nil
}

func provisionRole(
	ctx context.Context,
	rol role.Role,
	policies []policy.Policy,
	tx gorp.Tx,
	service *rbac.Service,
) (uuid.UUID, error) {
	policyKeys := make([]uuid.UUID, 0, len(policies))

	// Create or retrieve all policies
	for i := range policies {
		pol := &policies[i]
		if err := service.Policy.NewRetrieve().
			WhereNames(pol.Name).
			Entry(pol).
			Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
			return uuid.Nil, err
		}
		if pol.Key == uuid.Nil {
			if err := service.Policy.NewWriter(tx, true).Create(ctx, pol); err != nil {
				return uuid.Nil, err
			}
		}
		policyKeys = append(policyKeys, pol.Key)
	}

	// Create or retrieve the role
	if err := service.Role.NewRetrieve().
		WhereName(rol.Name).
		Entry(&rol).
		Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
		return uuid.Nil, err
	}
	if rol.Key == uuid.Nil {
		w := service.Role.NewWriter(tx, true)
		if err := w.Create(ctx, &rol); err != nil {
			return uuid.Nil, err
		}
		// Associate all policies with the role
		if err := service.Policy.NewWriter(tx, true).SetOnRole(ctx, rol.Key, policyKeys...); err != nil {
			return uuid.Nil, err
		}
	}
	return rol.Key, nil
}
