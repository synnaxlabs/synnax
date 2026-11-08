// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

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

// RolesMigration adds each stored role to the roles of the stored racks that a parent
// edge from the role points at, then deletes every parent edge from a role to a rack.
// An edge from a deleted role adds nothing.
var RolesMigration = ontology.NewFieldMigration(ontology.FieldConfig[Key, Rack]{
	Name: "v61_rack_roles",
	Converts: func(rel ontology.Relationship) bool {
		return rel.Type == ontology.RelationshipTypeParentOf &&
			rel.From.Type == resource.TypeRole &&
			rel.To.Type == resource.TypeRack
	},
	Holder: func(rel ontology.Relationship) string { return rel.To.Key },
	Set:    addRole,
})

func addRole(
	ctx context.Context,
	tx gorp.Tx,
	r *Rack,
	rel ontology.Relationship,
) (bool, error) {
	k, err := uuid.Parse(rel.From.Key)
	if err != nil {
		return false, errors.Wrapf(err, "role of edge %s", rel.GorpKey())
	}
	if slices.Contains(r.Roles, k) {
		return false, nil
	}
	exists, err := gorp.NewRetrieve[role.Key, role.Role]().
		Where(gorp.MatchKeys[role.Key, role.Role](k)).
		Exists(ctx, tx)
	if err != nil || !exists {
		return false, err
	}
	r.Roles = append(r.Roles, k)
	return true, nil
}
