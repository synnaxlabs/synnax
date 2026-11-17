// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v2 "github.com/synnaxlabs/synnax/pkg/service/legacy/device/versions/v2"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func parentOf(from, to resource.ID) ontology.Relationship {
	return ontology.Relationship{
		From: from,
		Type: ontology.RelationshipTypeParentOf,
		To:   to,
	}
}

func deviceID(key string) resource.ID {
	return resource.ID{Type: resource.TypeDevice, Key: key}
}

var _ = Describe("ParentMigration", func() {
	var (
		db       *gorp.DB
		rackID   = resource.ID{Type: resource.TypeRack, Key: "65537"}
		toTask   = parentOf(rackID, resource.ID{Type: resource.TypeTask, Key: "1"})
		devices  map[v2.Key]v2.Device
		retained []ontology.Relationship
	)

	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		stored := []v2.Device{
			{Key: "chassis", Rack: 65537},
			{Key: "module", Rack: 65537},
			{Key: "standalone", Rack: 65537},
			{Key: "orphan", Rack: 65537},
			{Key: "self", Rack: 65537},
		}
		Expect(gorp.NewCreate[v2.Key, v2.Device]().
			Entries(&stored).
			Exec(ctx, db)).To(Succeed())
		edges := []ontology.Relationship{
			parentOf(rackID, deviceID("chassis")),
			parentOf(deviceID("chassis"), deviceID("module")),
			parentOf(rackID, deviceID("standalone")),
			parentOf(deviceID("missing"), deviceID("orphan")),
			parentOf(deviceID("self"), deviceID("self")),
			toTask,
		}
		Expect(gorp.NewCreate[string, ontology.Relationship]().
			Entries(&edges).
			Exec(ctx, db)).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "device",
			Migrations: []gorp.Migration{v2.ParentMigration},
		})).To(Succeed())
		var out []v2.Device
		Expect(gorp.NewRetrieve[v2.Key, v2.Device]().Entries(&out).Exec(ctx, db)).
			To(Succeed())
		devices = make(map[v2.Key]v2.Device, len(out))
		for _, d := range out {
			devices[d.Key] = d
		}
		retained = nil
		Expect(gorp.NewRetrieve[string, ontology.Relationship]().
			Entries(&retained).
			Exec(ctx, db)).To(Succeed())
	})

	It("Should set the parent of a device held by another device", func() {
		Expect(devices["module"].Parent).To(HaveValue(Equal(v2.Key("chassis"))))
	})

	DescribeTable("Should leave no parent on a device",
		func(key v2.Key) { Expect(devices[key].Parent).To(BeNil()) },
		Entry("on its rack", v2.Key("chassis")),
		Entry("parented to a missing device", v2.Key("orphan")),
		Entry("parented to itself", v2.Key("self")),
	)

	It("Should delete every parent edge into a device", func() {
		Expect(retained).To(ConsistOf(toTask))
	})
})
