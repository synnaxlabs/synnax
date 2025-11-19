// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack_test

import (
	"slices"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/rack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Rack", Ordered, func() {
	var (
		db  *gorp.DB
		svc *rack.Service
		w   rack.Writer
		tx  gorp.Tx
	)
	BeforeAll(func() {
		db = gorp.Wrap(memkv.New())
		otg := MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
		g := MustSucceed(group.OpenService(ctx, group.Config{DB: db, Ontology: otg}))
		svc = MustSucceed(rack.OpenService(ctx, rack.Config{
			DB:           db,
			Ontology:     otg,
			Group:        g,
			HostProvider: mock.StaticHostKeyProvider(1),
		}))
	})
	BeforeEach(func() {
		tx = db.OpenTx()
		w = svc.NewWriter(tx)
	})
	AfterEach(func() {
		Expect(tx.Close()).To(Succeed())
	})
	Describe("Key", func() {
		It("Should correctly construct and deconstruct key from its components", func() {
			k := rack.NewKey(1, 2)
			Expect(k.Node()).To(Equal(cluster.NodeKey(1)))
			Expect(k.LocalKey()).To(Equal(uint16(2)))
		})
	})
	Describe("Create", func() {
		It("Should create a rack and assign it a key", func() {
			r := &rack.Rack{Name: "rack1"}
			Expect(w.Create(ctx, r)).To(Succeed())
			Expect(!r.Key.IsZero()).To(BeTrue())
			Expect(r.Key.Node()).To(Equal(cluster.NodeKey(1)))
			Expect(r.Key.LocalKey()).To(Equal(uint16(2)))
		})
		It("Should correctly increment the local key counter", func() {
			r := &rack.Rack{Name: "rack2"}
			Expect(w.Create(ctx, r)).To(Succeed())
			Expect(r.Key.LocalKey()).To(Equal(uint16(3)))
		})
		It("Should return an error if the rack has no name", func() {
			r := &rack.Rack{}
			err := w.Create(ctx, r)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("name: required"))
		})
	})
	Describe("Retrieve", func() {
		It("Should retrieve a rack by its key", func() {
			r := &rack.Rack{Name: "rack3"}
			Expect(w.Create(ctx, r)).To(Succeed())
			var res rack.Rack
			Expect(svc.NewRetrieve().WhereKeys(r.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal(*r))
		})
		It("Should retrieve racks where the host is the rack's node", func() {
			r := &rack.Rack{Name: "rack4"}
			Expect(w.Create(ctx, r)).To(Succeed())
			var res rack.Rack
			Expect(svc.NewRetrieve().WhereNodeIsHost(true).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal(*r))
		})
		It("Should only retrieve embedded racks", func() {
			var res rack.Rack
			Expect(svc.NewRetrieve().WhereEmbedded(true).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Embedded).To(BeTrue())
		})
	})
	Describe("DeleteChannel", func() {
		It("Should delete a rack by its key", func() {
			r := &rack.Rack{Name: "rack4"}
			Expect(w.Create(ctx, r)).To(Succeed())
			Expect(w.Delete(ctx, r.Key)).To(Succeed())
			var res rack.Rack
			Expect(svc.NewRetrieve().WhereKeys(r.Key).Entry(&res).Exec(ctx, tx)).To(MatchError(query.NotFound))
		})
	})

	Describe("Embedded Rack", func() {
		It("Should correctly create the node embedded rack", func() {
			Expect(svc.EmbeddedKey).ToNot(Equal(rack.Key(0)))
			var embeddedRack rack.Rack
			Expect(svc.NewRetrieve().WhereKeys(svc.EmbeddedKey).Entry(&embeddedRack).Exec(ctx, tx)).To(Succeed())
			Expect(embeddedRack.Embedded).To(BeTrue())
		})
	})

	Describe("NewTaskKey", func() {
		It("Should correctly return sequential keys", func() {
			r := &rack.Rack{Name: "niceRack"}
			w := svc.NewWriter(nil)
			Expect(w.Create(ctx, r)).To(Succeed())
			t1 := MustSucceed(svc.NewWriter(nil).NewTaskKey(ctx, r.Key))
			t2 := MustSucceed(svc.NewWriter(nil).NewTaskKey(ctx, r.Key))
			Expect(t2 - t1).To(BeEquivalentTo(1))
		})

		It("Should return sequential keys even when racing", func() {
			var (
				r     = &rack.Rack{Name: "niceRack"}
				w     = svc.NewWriter(nil)
				count = 100
				keys  = make([]uint32, count)
				wg    sync.WaitGroup
			)
			Expect(w.Create(ctx, r)).To(Succeed())

			for i := range count {
				wg.Go(func() {
					keys[i] = MustSucceed(svc.NewWriter(nil).NewTaskKey(ctx, r.Key))
				})
			}
			wg.Wait()

			slices.Sort(keys)
			for i := range keys {
				if i == 0 {
					continue
				}
				Expect(keys[i] - keys[i-1]).To(BeEquivalentTo(1))
			}

		})
	})
})

var _ = Describe("Migration", func() {
	It("Should correctly migrate a v1 rack to a v2 rack", func() {
		db := gorp.Wrap(memkv.New())
		otg := MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
		g := MustSucceed(group.OpenService(ctx, group.Config{DB: db, Ontology: otg}))

		v1EmbeddedRack := rack.Rack{
			Key:  65538,
			Name: "sy_node_1_rack",
		}
		Expect(gorp.NewCreate[rack.Key, rack.Rack]().
			Entry(&v1EmbeddedRack).
			Exec(ctx, db)).To(Succeed())

		svc := MustSucceed(rack.OpenService(ctx, rack.Config{
			DB:           db,
			Ontology:     otg,
			Group:        g,
			HostProvider: mock.StaticHostKeyProvider(1),
		}))
		Expect(svc.EmbeddedKey).To(Equal(rack.Key(65538)))
		// Retrieve the embedded rack
		var embeddedRack rack.Rack
		Expect(svc.NewRetrieve().WhereKeys(svc.EmbeddedKey).Entry(&embeddedRack).Exec(ctx, db)).To(Succeed())
		Expect(embeddedRack.Embedded).To(BeTrue())
		Expect(embeddedRack.Name).To(Equal("Node 1 Embedded Driver"))
		count := MustSucceed(gorp.NewRetrieve[rack.Key, rack.Rack]().Count(ctx, db))
		Expect(count).To(Equal(1))
	})
})
