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

	"github.com/synnaxlabs/gorp"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/set"
)

// The reasons a migration removes a member from a group.
const (
	// RemovedDuplicate is a member of several groups that stays only in the group
	// that comes first by name.
	RemovedDuplicate ontology.RelationshipType = "v61_removed_duplicate"
	// RemovedUnheld is a member of a group whose scope cannot hold its type.
	RemovedUnheld ontology.RelationshipType = "v61_removed_unheld"
	// RemovedPinned is a project or rack in a group under a project or rack.
	RemovedPinned ontology.RelationshipType = "v61_removed_pinned"
	// RemovedWithGroup is a member of a group that the upgrade deletes.
	RemovedWithGroup ontology.RelationshipType = "v61_removed_with_group"
	// RemovedRackless is a device whose rack does not exist.
	RemovedRackless ontology.RelationshipType = "v61_removed_rackless"
)

var reasons = set.New(
	RemovedDuplicate,
	RemovedUnheld,
	RemovedPinned,
	RemovedWithGroup,
	RemovedRackless,
)

// removedFromType is the type of the source of a staged removal edge. Its key is the
// name of the group, since a later migration can delete the group.
const removedFromType resource.Type = "v61_removed_from"

// Removal is a member that a migration removed from a group.
type Removal struct {
	// GroupName is the name of the group, as it was when the member was removed.
	GroupName string
	// Member is the removed member.
	Member resource.ID
	// Reason is why the member was removed.
	Reason ontology.RelationshipType
}

// StageRemovals records removals in the ontology, where RetrieveRemovals reads them
// until the ontology is dropped. Later migrations that rewrite the keys of ontology
// edges rewrite the members of staged removals too.
func StageRemovals(ctx context.Context, tx gorp.Tx, removals ...Removal) error {
	rels := make([]ontology.Relationship, len(removals))
	for i, r := range removals {
		rels[i] = ontology.Relationship{
			From: resource.ID{Type: removedFromType, Key: r.GroupName},
			Type: r.Reason,
			To:   r.Member,
		}
	}
	return gorp.WrapWriter[string, ontology.Relationship](tx).Set(ctx, rels...)
}

// RetrieveRemovals returns the removals that StageRemovals recorded.
func RetrieveRemovals(ctx context.Context, tx gorp.Tx) ([]Removal, error) {
	rels, err := ontology.RetrieveRelationships(
		ctx,
		tx,
		func(r ontology.Relationship) bool {
			return r.From.Type == removedFromType && reasons.Contains(r.Type)
		},
	)
	if err != nil {
		return nil, err
	}
	out := make([]Removal, len(rels))
	for i, r := range rels {
		out[i] = Removal{GroupName: r.From.Key, Member: r.To, Reason: r.Type}
	}
	return out, nil
}
