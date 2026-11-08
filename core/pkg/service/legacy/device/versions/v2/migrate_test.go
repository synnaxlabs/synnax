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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/legacy/device/versions/v2"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	rack "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("MembersMigration", func() {
	It("Should keep a device only in a group under its rack", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		r := rack.Key(65538)
		under := groupv1.Group{
			Key:    uuid.New(),
			Name:   "Under rack",
			Scope:  resource.TypeDevice,
			Parent: new(r.ResourceID()),
		}
		loose := groupv1.Group{
			Key:   uuid.New(),
			Name:  "Loose",
			Scope: resource.TypeDevice,
		}
		groups := []groupv1.Group{under, loose}
		Expect(gorp.NewCreate[groupv1.Key, groupv1.Group]().
			Entries(&groups).
			Exec(ctx, db)).To(Succeed())
		devices := []v2.Device{
			{Key: "kept", Rack: r, Name: "Kept"},
			{Key: "dropped", Rack: r, Name: "Dropped"},
		}
		Expect(gorp.NewCreate[v2.Key, v2.Device]().Entries(&devices).Exec(ctx, db)).
			To(Succeed())
		edges := []ontology.Relationship{
			{
				From: resource.ID{Type: resource.TypeGroup, Key: under.Key.String()},
				Type: ontology.RelationshipTypeParentOf,
				To:   devices[0].ResourceID(),
			},
			{
				From: resource.ID{Type: resource.TypeGroup, Key: loose.Key.String()},
				Type: ontology.RelationshipTypeParentOf,
				To:   devices[1].ResourceID(),
			},
		}
		Expect(gorp.NewCreate[string, ontology.Relationship]().
			Entries(&edges).
			Exec(ctx, db)).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "device",
			Migrations: []gorp.Migration{v2.MembersMigration},
		})).To(Succeed())
		var out []v2.Device
		Expect(gorp.NewRetrieve[v2.Key, v2.Device]().
			Where(gorp.MatchKeys[v2.Key, v2.Device]("kept", "dropped")).
			Entries(&out).
			Exec(ctx, db)).To(Succeed())
		groupOf := make(map[v2.Key]*groupv1.Key, len(out))
		for _, d := range out {
			groupOf[d.Key] = d.Group
		}
		Expect(groupOf).To(HaveKeyWithValue("kept", HaveValue(Equal(under.Key))))
		Expect(groupOf).To(HaveKeyWithValue("dropped", BeNil()))
	})
})
