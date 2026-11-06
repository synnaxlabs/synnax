// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil grants access through RBAC for specs.
package testutil

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
)

// Grant gives the user with the given key a new role whose one policy allows the
// actions on the objects, and returns the key of the role.
func Grant(
	ctx context.Context,
	rbacSvc *rbac.Service,
	users *user.Service,
	key user.Key,
	actions []access.Action,
	objects ...resource.ID,
) (role.Key, error) {
	r := role.Role{Name: "grant-" + uuid.New().String()}
	if err := rbacSvc.Role.NewWriter(nil, false).Create(ctx, &r); err != nil {
		return uuid.Nil(), err
	}
	if err := rbacSvc.Policy.NewWriter(nil, false).Create(ctx, &policy.Policy{
		Role:    r.Key,
		Name:    r.Name,
		Objects: objects,
		Actions: actions,
	}); err != nil {
		return uuid.Nil(), err
	}
	return r.Key, users.NewWriter(nil).AssignRole(ctx, r.Key, key)
}
