// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resource_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

type target struct{ Key string }

func (t target) GorpKey() string { return t.Key }

// dependent uses one target and is placed under another.
type dependent struct {
	Key    string
	Uses   string
	Parent string
}

func (d dependent) GorpKey() string { return d.Key }

func optional(key string) []string {
	if key == "" {
		return nil
	}
	return []string{key}
}

var _ = Describe("Reference", func() {
	var (
		db         *gorp.DB
		targets    *gorp.Table[string, target]
		dependents *gorp.Table[string, dependent]
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		targets = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, target]{
			DB:   db,
			Name: "target",
		}))
		dependents = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, dependent]{
			DB:   db,
			Name: "dependent",
			References: []gorp.Reference[string, dependent]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, dependent, string]{
					Name:     "uses",
					Table:    "target",
					OnDelete: gorp.Restrict,
					Kind:     gorp.Use,
					Targets:  func(d *dependent) []string { return optional(d.Uses) },
				}),
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, dependent, string]{
					Name:     "parent",
					Table:    "target",
					OnDelete: gorp.Cascade,
					Kind:     gorp.Place,
					Targets:  func(d *dependent) []string { return optional(d.Parent) },
				}),
			},
		}))
		Expect(targets.NewCreate().Entry(&target{Key: "t"}).Exec(ctx, db)).
			To(Succeed())
		Expect(dependents.NewCreate().
			Entries(&[]dependent{{Key: "user", Uses: "t"}, {Key: "child", Parent: "t"}}).
			Exec(ctx, db)).To(Succeed())
	})

	Describe("Dependents", func() {
		It("Should list every record that references the entry", func(ctx SpecContext) {
			Expect(resource.Dependents(ctx, db, targets, "t")).To(ConsistOf(
				resource.ID{Type: "dependent", Key: "user"},
				resource.ID{Type: "dependent", Key: "child"},
			))
		})
	})

	Describe("Uses", func() {
		It("Should list only the records that use the entry", func(ctx SpecContext) {
			Expect(resource.Uses(ctx, db, targets, "t")).
				To(Equal([]resource.ID{{Type: "dependent", Key: "user"}}))
		})
	})
})
