// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"
	"slices"
	"uuid"

	"github.com/synnaxlabs/gorp"
	role "github.com/synnaxlabs/synnax/pkg/service/access/rbac/role/versions/v0"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
)

// RolesMigration adds the role of each parent edge from a role to a user to the roles
// of the user, after the roles it holds, then deletes every such edge. An edge from a
// deleted role adds nothing.
var RolesMigration = ontology.NewFieldMigration(ontology.FieldConfig[Key, User]{
	Name: "v61_user_roles",
	Converts: func(rel ontology.Relationship) bool {
		return rel.Type == ontology.RelationshipTypeParentOf &&
			rel.From.Type == resource.TypeRole &&
			rel.To.Type == resource.TypeUser
	},
	Holder: func(rel ontology.Relationship) string { return rel.To.Key },
	Set:    addRole,
})

func addRole(
	ctx context.Context,
	tx gorp.Tx,
	u *User,
	rel ontology.Relationship,
) (bool, error) {
	r, err := uuid.Parse(rel.From.Key)
	if err != nil {
		return false, errors.Wrapf(err, "role of edge %s", rel.GorpKey())
	}
	if slices.Contains(u.Roles, r) {
		return false, nil
	}
	exists, err := gorp.NewRetrieve[role.Key, role.Role]().
		Where(gorp.MatchKeys[role.Key, role.Role](r)).
		Exists(ctx, tx)
	if err != nil || !exists {
		return false, err
	}
	u.Roles = append(u.Roles, r)
	return true, nil
}
