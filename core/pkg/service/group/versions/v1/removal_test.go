// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Removal", func() {
	It("Should stage a removal from each of two groups with the same name",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			StartRun(db.Run)
			member := resource.ID{Type: resource.TypeChannel, Key: "1"}
			first := v1.Removal{
				Group:     uuid.New(),
				GroupName: "Sensors/A",
				Member:    member,
				Reason:    v1.RemovedDuplicate,
			}
			second := first
			second.Group = uuid.New()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return v1.StageRemovals(ctx, tx, first, second)
			})).To(Succeed())
			createEdges(ctx, db,
				parentOf(groupID(first.Group), member),
				ontology.Relationship{
					From: groupID(first.Group),
					Type: "v61_unknown",
					To:   member,
				},
			)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				Expect(v1.RetrieveRemovals(ctx, tx)).To(ConsistOf(first, second))
				return nil
			})).To(Succeed())
		},
	)

	DescribeTable("Should describe why the member left the group",
		func(reason ontology.RelationshipType, noun, description string) {
			r := v1.Removal{GroupName: "Bench", Reason: reason}
			Expect(r.Describe(noun, "pump")).To(Equal(description))
		},
		Entry("a duplicate", v1.RemovedDuplicate, "channel",
			`A channel can now be in only one group, so the upgrade removed channel `+
				`"pump" from group "Bench". It kept the channel in the group that `+
				`comes first by name.`),
		Entry("a member the group cannot hold", v1.RemovedUnheld, "channel",
			`The upgrade removed channel "pump" from group "Bench", because that `+
				`group cannot hold a channel.`),
		Entry("a pinned member", v1.RemovedPinned, "rack",
			`The upgrade removed rack "pump" from group "Bench", because a rack `+
				`cannot be in a group inside a project or rack.`),
		Entry("a member of a deleted group", v1.RemovedWithGroup, "schematic",
			`The upgrade deleted group "Bench", so schematic "pump" is no longer in `+
				`it.`),
		Entry("a device with no rack", v1.RemovedRackless, "device",
			`The upgrade removed device "pump" from group "Bench", because the rack `+
				`of the device does not exist.`),
	)
})
