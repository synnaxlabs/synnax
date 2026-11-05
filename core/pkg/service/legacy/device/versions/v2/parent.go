// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"context"

	"github.com/synnaxlabs/gorp"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// ParentMigration sets the parent of each stored device from the parent edge that
// points at it from another stored device. It then deletes every parent edge into a
// device from a rack or a device.
var ParentMigration = ontology.NewFieldMigration(ontology.FieldConfig[Key, Device]{
	Name: "v61_device_parent",
	Converts: func(rel ontology.Relationship) bool {
		return rel.Type == ontology.RelationshipTypeParentOf &&
			rel.To.Type == resource.TypeDevice &&
			(rel.From.Type == resource.TypeRack || rel.From.Type == resource.TypeDevice)
	},
	Holder: func(rel ontology.Relationship) string { return rel.To.Key },
	Set:    setParent,
})

func setParent(
	ctx context.Context,
	tx gorp.Tx,
	d *Device,
	rel ontology.Relationship,
) (bool, error) {
	if rel.From.Type != resource.TypeDevice || rel.From.Key == rel.To.Key ||
		d.Parent != nil {
		return false, nil
	}
	parent := Key(rel.From.Key)
	exists, err := gorp.NewRetrieve[Key, Device]().
		Where(gorp.MatchKeys[Key, Device](parent)).
		Exists(ctx, tx)
	if err != nil || !exists {
		return false, err
	}
	d.Parent = &parent
	return true, nil
}
