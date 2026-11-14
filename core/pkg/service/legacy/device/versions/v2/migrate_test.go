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
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/synnaxlabs/gorp"
	groupv0 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v0"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/legacy/device/versions/v2"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	rack "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var rootID = resource.ID{Type: resource.TypeBuiltin, Key: "root"}

func groupID(k uuid.UUID) resource.ID {
	return resource.ID{Type: resource.TypeGroup, Key: k.String()}
}

var _ = Describe("Group migrations", func() {
	var (
		db      *gorp.DB
		r1      = rack.Key(65537)
		r2      = rack.Key(65538)
		devices = uuid.New()
		inRack  = uuid.New()
		loose   = uuid.New()
		nested  = uuid.New()
		racks   = uuid.New()
		first   = uuid.MustParse("00000000-0000-0000-0000-000000000001")
		second  = uuid.MustParse("00000000-0000-0000-0000-000000000002")
		both    = []rack.Rack{{Key: r1, Name: "Rack 1"}, {Key: r2, Name: "Rack 2"}}
	)

	// migrate creates a 0.60.1 store of the given groups, racks, devices, and parent
	// edges, then runs the group and device migrations on it.
	migrate := func(
		ctx context.Context,
		names map[uuid.UUID]string,
		rackRows []rack.Rack,
		deviceRows []v2.Device,
		edges ...ontology.Relationship,
	) {
		GinkgoHelper()
		db = DeferClose(gorp.Wrap(memkv.New()))
		for k, name := range names {
			g := groupv0.Group{Key: k, Name: name}
			Expect(gorp.NewCreate[groupv0.Key, groupv0.Group]().
				Entry(&g).
				Exec(ctx, db)).To(Succeed())
		}
		Expect(gorp.NewCreate[string, ontology.Relationship]().
			Entries(&edges).
			Exec(ctx, db)).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "group",
			Migrations: []gorp.Migration{groupv1.Migration, groupv1.ParentMigration},
		})).To(Succeed())
		Expect(gorp.NewCreate[rack.Key, rack.Rack]().
			Entries(&rackRows).
			Exec(ctx, db)).To(Succeed())
		Expect(gorp.NewCreate[v2.Key, v2.Device]().
			Entries(&deviceRows).
			Exec(ctx, db)).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "device",
			Migrations: []gorp.Migration{
				v2.SplitGroupsMigration,
				v2.MembersMigration,
			},
		})).To(Succeed())
	}

	retrieveGroups := func(ctx context.Context) map[groupv1.Key]groupv1.Group {
		GinkgoHelper()
		var out []groupv1.Group
		Expect(gorp.NewRetrieve[groupv1.Key, groupv1.Group]().
			Entries(&out).
			Exec(ctx, db)).To(Succeed())
		byKey := make(map[groupv1.Key]groupv1.Group, len(out))
		for _, g := range out {
			byKey[g.Key] = g
		}
		return byKey
	}

	retrieveDevice := func(ctx context.Context, key v2.Key) v2.Device {
		GinkgoHelper()
		var d v2.Device
		Expect(gorp.NewRetrieve[v2.Key, v2.Device]().
			Where(gorp.MatchKeys[v2.Key, v2.Device](key)).
			Entry(&d).
			Exec(ctx, db)).To(Succeed())
		return d
	}

	device := func(key v2.Key, r rack.Key) v2.Device {
		return v2.Device{Key: key, Rack: r, Name: key}
	}

	It("Should keep a device in a group under its rack", func(ctx SpecContext) {
		migrate(
			ctx,
			map[uuid.UUID]string{inRack: "Under rack"},
			both,
			[]v2.Device{device("a", r1)},
			parentOf(r1.ResourceID(), groupID(inRack)),
			parentOf(groupID(inRack), deviceID("a")),
		)
		Expect(retrieveDevice(ctx, "a").Group).To(HaveValue(Equal(inRack)))
		Expect(retrieveGroups(ctx)).To(HaveKey(inRack))
	})

	It("Should split a group under the Devices root by rack", func(ctx SpecContext) {
		migrate(
			ctx,
			map[uuid.UUID]string{devices: "Devices", loose: "Thermocouples"},
			both,
			[]v2.Device{device("a", r1), device("b", r2), device("c", r1)},
			parentOf(rootID, groupID(devices)),
			parentOf(groupID(devices), groupID(loose)),
			parentOf(groupID(loose), deviceID("a")),
			parentOf(groupID(loose), deviceID("b")),
			parentOf(groupID(loose), deviceID("c")),
		)
		groups := retrieveGroups(ctx)
		Expect(groups).ToNot(HaveKey(loose))
		a, b, c := retrieveDevice(ctx, "a"), retrieveDevice(ctx, "b"),
			retrieveDevice(ctx, "c")
		Expect(c.Group).To(Equal(a.Group))
		Expect(groups[*a.Group]).To(MatchFields(IgnoreExtras, Fields{
			"Name":   Equal("Thermocouples"),
			"Scope":  Equal(resource.TypeDevice),
			"Parent": HaveValue(Equal(r1.ResourceID())),
		}))
		Expect(groups[*b.Group]).To(MatchFields(IgnoreExtras, Fields{
			"Name":   Equal("Thermocouples"),
			"Scope":  Equal(resource.TypeDevice),
			"Parent": HaveValue(Equal(r2.ResourceID())),
		}))
	})

	It("Should copy a nested chain under the rack", func(ctx SpecContext) {
		migrate(
			ctx,
			map[uuid.UUID]string{devices: "Devices", loose: "Outer", nested: "Inner"},
			both,
			[]v2.Device{device("a", r1)},
			parentOf(rootID, groupID(devices)),
			parentOf(groupID(devices), groupID(loose)),
			parentOf(groupID(loose), groupID(nested)),
			parentOf(groupID(nested), deviceID("a")),
		)
		groups := retrieveGroups(ctx)
		Expect(groups).ToNot(HaveKey(loose))
		Expect(groups).ToNot(HaveKey(nested))
		inner := groups[*retrieveDevice(ctx, "a").Group]
		Expect(inner.Name).To(Equal("Inner"))
		Expect(inner.Parent).ToNot(BeNil())
		outer := groups[uuid.MustParse(inner.Parent.Key)]
		Expect(outer.Name).To(Equal("Outer"))
		Expect(outer.Parent).To(HaveValue(Equal(r1.ResourceID())))
	})

	It("Should move a device out of a group under another rack", func(ctx SpecContext) {
		migrate(
			ctx,
			map[uuid.UUID]string{inRack: "Sensors"},
			both,
			[]v2.Device{device("a", r1), device("b", r2)},
			parentOf(r1.ResourceID(), groupID(inRack)),
			parentOf(groupID(inRack), deviceID("a")),
			parentOf(groupID(inRack), deviceID("b")),
		)
		groups := retrieveGroups(ctx)
		Expect(retrieveDevice(ctx, "a").Group).To(HaveValue(Equal(inRack)))
		b := retrieveDevice(ctx, "b")
		Expect(groups[*b.Group]).To(MatchFields(IgnoreExtras, Fields{
			"Name":   Equal("Sensors"),
			"Parent": HaveValue(Equal(r2.ResourceID())),
		}))
	})

	It("Should keep a split group that holds a rack", func(ctx SpecContext) {
		migrate(
			ctx,
			map[uuid.UUID]string{devices: "Devices", racks: "Racks"},
			[]rack.Rack{{Key: r1, Name: "Rack", Group: new(racks)}},
			[]v2.Device{device("a", r1)},
			parentOf(rootID, groupID(devices)),
			parentOf(groupID(devices), groupID(racks)),
			parentOf(groupID(racks), deviceID("a")),
		)
		groups := retrieveGroups(ctx)
		Expect(groups).To(HaveKey(racks))
		a := retrieveDevice(ctx, "a")
		Expect(*a.Group).ToNot(Equal(racks))
		Expect(groups[*a.Group].Parent).To(HaveValue(Equal(r1.ResourceID())))
	})

	It("Should copy only the first group of a device", func(ctx SpecContext) {
		migrate(
			ctx,
			map[uuid.UUID]string{devices: "Devices", first: "First", second: "Second"},
			both,
			[]v2.Device{device("a", r1)},
			parentOf(rootID, groupID(devices)),
			parentOf(groupID(devices), groupID(first)),
			parentOf(groupID(devices), groupID(second)),
			parentOf(groupID(first), deviceID("a")),
			parentOf(groupID(second), deviceID("a")),
		)
		groups := retrieveGroups(ctx)
		Expect(groups).ToNot(HaveKey(first))
		Expect(groups).To(HaveKey(second))
		Expect(groups).To(HaveLen(3))
		Expect(
			groups[*retrieveDevice(ctx, "a").Group],
		).To(MatchFields(IgnoreExtras, Fields{
			"Name":   Equal("First"),
			"Parent": HaveValue(Equal(r1.ResourceID())),
		}))
	})

	It(
		"Should keep a device in a first group under its rack",
		func(ctx SpecContext) {
			migrate(
				ctx,
				map[uuid.UUID]string{
					devices: "Devices",
					first:   "Rack",
					second:  "Loose",
				},
				both,
				[]v2.Device{device("a", r1)},
				parentOf(r1.ResourceID(), groupID(first)),
				parentOf(rootID, groupID(devices)),
				parentOf(groupID(devices), groupID(second)),
				parentOf(groupID(first), deviceID("a")),
				parentOf(groupID(second), deviceID("a")),
			)
			Expect(retrieveDevice(ctx, "a").Group).To(HaveValue(Equal(first)))
			Expect(retrieveGroups(ctx)).To(HaveLen(3))
		},
	)

	It("Should drop a device whose rack does not exist", func(ctx SpecContext) {
		migrate(
			ctx,
			map[uuid.UUID]string{devices: "Devices", loose: "Loose"},
			both,
			[]v2.Device{device("a", rack.Key(65539))},
			parentOf(rootID, groupID(devices)),
			parentOf(groupID(devices), groupID(loose)),
			parentOf(groupID(loose), deviceID("a")),
		)
		Expect(retrieveDevice(ctx, "a").Group).To(BeNil())
		Expect(retrieveGroups(ctx)).To(HaveLen(2))
	})

	It("Should keep a split group that holds a child group", func(ctx SpecContext) {
		migrate(
			ctx,
			map[uuid.UUID]string{devices: "Devices", loose: "Outer", nested: "Inner"},
			both,
			[]v2.Device{device("a", r1)},
			parentOf(rootID, groupID(devices)),
			parentOf(groupID(devices), groupID(loose)),
			parentOf(groupID(loose), groupID(nested)),
			parentOf(groupID(loose), deviceID("a")),
		)
		groups := retrieveGroups(ctx)
		Expect(groups).To(HaveKey(loose))
		Expect(groups).To(HaveKey(nested))
		Expect(
			groups[*retrieveDevice(ctx, "a").Group],
		).To(MatchFields(IgnoreExtras, Fields{
			"Name":   Equal("Outer"),
			"Parent": HaveValue(Equal(r1.ResourceID())),
		}))
	})
})

var _ = Describe("KafkaSecureMigration", func() {
	migrateDevice := func(ctx SpecContext, d v2.Device) v2.Device {
		GinkgoHelper()
		db := DeferClose(gorp.Wrap(memkv.New()))
		Expect(gorp.NewCreate[v2.Key, v2.Device]().Entry(&d).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Device",
			Migrations: []gorp.Migration{v2.KafkaSecureMigration},
		})).To(Succeed())
		var got v2.Device
		Expect(gorp.NewRetrieve[v2.Key, v2.Device]().
			Where(gorp.MatchKeys[v2.Key, v2.Device](d.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		return got
	}

	It("Should rename the tls property of a Kafka device to secure", func(
		ctx SpecContext,
	) {
		got := migrateDevice(ctx, v2.Device{
			Key:        "kafka-1",
			Make:       "kafka",
			Properties: msgpack.EncodedJSON{"brokers": []any{"b:9092"}, "tls": true},
		})
		Expect(got.Properties).To(HaveKeyWithValue("secure", true))
		Expect(got.Properties).NotTo(HaveKey("tls"))
		Expect(got.Properties).To(HaveKey("brokers"))
	})

	It("Should leave the tls property of another make", func(ctx SpecContext) {
		got := migrateDevice(ctx, v2.Device{
			Key:        "other-1",
			Make:       "other",
			Properties: msgpack.EncodedJSON{"tls": true},
		})
		Expect(got.Properties).To(HaveKeyWithValue("tls", true))
		Expect(got.Properties).NotTo(HaveKey("secure"))
	})
})
