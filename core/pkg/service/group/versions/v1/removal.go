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
	"fmt"
	"strings"
	"uuid"

	"github.com/synnaxlabs/gorp"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// The reasons a migration removes a member from a group.
const (
	// RemovedDuplicate is a member of several groups that stays only in the group that
	// comes first by name.
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

// descriptions writes the description of a removal for each reason, given the member's
// type, the member's name, and the group's name.
var descriptions = map[ontology.RelationshipType]func(
	noun, member, group string,
) string{
	RemovedDuplicate: func(noun, member, group string) string {
		return fmt.Sprintf(
			"A %[1]s can now be in only one group, so the upgrade removed %[1]s %[2]q "+
				"from group %[3]q. It kept the %[1]s in the group that comes first by "+
				"name.",
			noun, member, group,
		)
	},
	RemovedUnheld: func(noun, member, group string) string {
		return fmt.Sprintf(
			"The upgrade removed %[1]s %[2]q from group %[3]q, because that group "+
				"cannot hold a %[1]s.",
			noun, member, group,
		)
	},
	RemovedPinned: func(noun, member, group string) string {
		return fmt.Sprintf(
			"The upgrade removed %[1]s %[2]q from group %[3]q, because a %[1]s cannot "+
				"be in a group inside a project or rack.",
			noun, member, group,
		)
	},
	RemovedWithGroup: func(noun, member, group string) string {
		return fmt.Sprintf(
			"The upgrade deleted group %[3]q, so %[1]s %[2]q is no longer in it.",
			noun, member, group,
		)
	},
	RemovedRackless: func(noun, member, group string) string {
		return fmt.Sprintf(
			"The upgrade removed %[1]s %[2]q from group %[3]q, because the rack of the "+
				"%[1]s does not exist.",
			noun,
			member,
			group,
		)
	},
}

// removedFromType is the type of the source of a staged removal edge. Its key is the
// key and the name of the group, since a later migration can delete the group.
const removedFromType resource.Type = "v61_removed_from"

// Removal is a member that a migration removed from a group.
type Removal struct {
	// Group is the key of the group.
	Group Key
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
			From: resource.ID{
				Type: removedFromType,
				Key:  r.Group.String() + "/" + r.GroupName,
			},
			Type: r.Reason,
			To:   r.Member,
		}
	}
	return gorp.WrapWriter[string, ontology.Relationship](tx).Set(ctx, rels...)
}

func isReason(t ontology.RelationshipType) bool {
	_, ok := descriptions[t]
	return ok
}

// Describe returns the description of the removal, given the member's type and name.
func (r Removal) Describe(noun, member string) string {
	return descriptions[r.Reason](noun, member, r.GroupName)
}

// RetrieveRemovals returns the removals that StageRemovals recorded.
func RetrieveRemovals(ctx context.Context, tx gorp.Tx) ([]Removal, error) {
	rels, err := ontology.RetrieveRelationships(
		ctx,
		tx,
		func(r ontology.Relationship) bool {
			return r.From.Type == removedFromType && isReason(r.Type)
		},
	)
	if err != nil {
		return nil, err
	}
	out := make([]Removal, len(rels))
	for i, r := range rels {
		key, name, _ := strings.Cut(r.From.Key, "/")
		g, err := uuid.Parse(key)
		if err != nil {
			return nil, err
		}
		out[i] = Removal{Group: g, GroupName: name, Member: r.To, Reason: r.Type}
	}
	return out, nil
}
