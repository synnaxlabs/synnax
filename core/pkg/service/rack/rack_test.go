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
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Rack", Ordered, func() {
	var (
		writer rack.Writer
		tx     gorp.Tx
		db     *gorp.DB
		svc    *rack.Service
		stat   *status.Service
	)

	BeforeAll(func() {
		db = gorp.Wrap(memkv.New())
		otg := MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
		g := MustSucceed(group.OpenService(ctx, group.Config{DB: db, Ontology: otg}))
		label := MustSucceed(label.OpenService(ctx, label.Config{
			DB:       db,
			Ontology: otg,
			Group:    g,
		}))
		stat = MustSucceed(status.OpenService(ctx, status.ServiceConfig{
			Ontology: otg,
			DB:       db,
			Group:    g,
			Label:    label,
		}))
		svc = MustSucceed(rack.OpenService(ctx, rack.Config{
			DB:                  db,
			Ontology:            otg,
			Group:               g,
			HostProvider:        mock.StaticHostKeyProvider(1),
			Status:              stat,
			HealthCheckInterval: 10 * telem.Millisecond,
		}))
		DeferCleanup(func() {
			Expect(svc.Close()).To(Succeed())
			Expect(stat.Close()).To(Succeed())
			Expect(label.Close()).To(Succeed())
			Expect(g.Close()).To(Succeed())
			Expect(otg.Close()).To(Succeed())
			Expect(db.Close()).To(Succeed())
		})
	})
	BeforeEach(func() {
		tx = db.OpenTx()
		writer = svc.NewWriter(tx)
		DeferCleanup(func() {
			Expect(tx.Close()).To(Succeed())
		})
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
			Expect(writer.Create(ctx, r)).To(Succeed())
			Expect(!r.Key.IsZero()).To(BeTrue())
			Expect(r.Key.Node()).To(Equal(cluster.NodeKey(1)))
			Expect(r.Key.LocalKey()).To(Equal(uint16(2)))
		})
		It("Should correctly increment the local key counter", func() {
			r := &rack.Rack{Name: "rack2"}
			Expect(writer.Create(ctx, r)).To(Succeed())
			Expect(r.Key.LocalKey()).To(Equal(uint16(3)))
		})
		It("Should return an error if the rack has no name", func() {
			r := &rack.Rack{}
			err := writer.Create(ctx, r)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("name: required"))
		})
	})
	Describe("Retrieve", func() {
		It("Should retrieve a rack by its key", func() {
			r := &rack.Rack{Name: "rack3"}
			Expect(writer.Create(ctx, r)).To(Succeed())
			var res rack.Rack
			Expect(svc.NewRetrieve().WhereKeys(r.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal(*r))
		})
		It("Should retrieve racks where the host is the rack's node", func() {
			r := &rack.Rack{Name: "rack4"}
			Expect(writer.Create(ctx, r)).To(Succeed())
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
			Expect(writer.Create(ctx, r)).To(Succeed())
			Expect(writer.Delete(ctx, r.Key)).To(Succeed())
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

	Describe("Status", func() {
		It("Should initialize a rack with an unknown status", func() {
			r := rack.Rack{Name: "test rack"}
			Expect(svc.NewWriter(nil).Create(ctx, &r)).To(Succeed())
			s := MustSucceed(svc.RetrieveStatus(ctx, r.Key))
			Expect(s.Message).To(Equal("Status unknown"))
			Expect(s.Variant).To(Equal(xstatus.WarningVariant))
			Expect(s.Time).To(BeNumerically("~", telem.Now(), 3*telem.SecondTS))
			Expect(s.Key).To(ContainSubstring(string(rack.OntologyType)))
			Expect(s.Details.Rack).To(Equal(r.Key))
		})

		It("Should use the provided status when creating a rack", func() {
			providedStatus := &rack.Status{
				Variant:     xstatus.SuccessVariant,
				Message:     "Custom status message",
				Description: "Custom description",
			}
			r := rack.Rack{Name: "rack with custom status", Status: providedStatus}
			Expect(svc.NewWriter(nil).Create(ctx, &r)).To(Succeed())
			s := MustSucceed(svc.RetrieveStatus(ctx, r.Key))
			Expect(s.Message).To(Equal("Custom status message"))
			Expect(s.Description).To(Equal("Custom description"))
			Expect(s.Variant).To(Equal(xstatus.SuccessVariant))
			// Key should be auto-assigned to match ontology ID
			Expect(s.Key).To(Equal(rack.OntologyID(r.Key).String()))
			// Time should be auto-filled
			Expect(s.Time).To(BeNumerically("~", telem.Now(), 3*telem.SecondTS))
			// Name should be auto-filled from rack name
			Expect(s.Name).To(Equal(r.Name))
			// Details.Rack should be auto-filled
			Expect(s.Details.Rack).To(Equal(r.Key))
		})

		It("Should return a validation error if provided status has empty variant", func() {
			providedStatus := &rack.Status{
				Message: "Status with no variant",
			}
			r := rack.Rack{Name: "rack with invalid status", Status: providedStatus}
			err := svc.NewWriter(nil).Create(ctx, &r)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("variant"))
		})

		It("Should mark a rack as dead when it doesn't receive a status within the health check interval", func() {
			r := rack.Rack{Name: "dead test rack"}
			Expect(svc.NewWriter(nil).Create(ctx, &r)).To(Succeed())

			Eventually(func(g Gomega) {
				s := MustSucceed(svc.RetrieveStatus(ctx, r.Key))
				g.Expect(s.Message).To(Equal("Synnax Driver on dead test rack not running"))
				g.Expect(s.Variant).To(Equal(xstatus.WarningVariant))
				g.Expect(s.Time).To(BeNumerically("~", telem.Now(), 3*telem.SecondTS))
				g.Expect(s.Key).To(ContainSubstring(string(rack.OntologyType)))
				g.Expect(s.Details.Rack).To(Equal(r.Key))
				g.Expect(s.Description).To(ContainSubstring("Driver was last alive"))
			}).Should(Succeed())
		})

		It("Should not mark a rack as dead when the status is actively updated", func() {
			r := rack.Rack{Name: "active test rack"}
			Expect(svc.NewWriter(nil).Create(ctx, &r)).To(Succeed())

			statusWriter := status.NewWriter[rack.StatusDetails](stat, nil)

			// Wait longer than the health check interval and verify the status
			// is still what we set (not overwritten with "not running") as long
			// as we keep actively updating it
			Consistently(func(g Gomega) {
				// Actively update the status to simulate a running driver
				activeStatus := &rack.Status{
					Key:     rack.OntologyID(r.Key).String(),
					Name:    r.Name,
					Time:    telem.Now(),
					Variant: xstatus.SuccessVariant,
					Message: "Running",
					Details: rack.StatusDetails{Rack: r.Key},
				}
				g.Expect(statusWriter.Set(ctx, activeStatus)).To(Succeed())

				s := MustSucceed(svc.RetrieveStatus(ctx, r.Key))
				g.Expect(s.Message).To(Equal("Running"))
				g.Expect(s.Variant).To(Equal(xstatus.SuccessVariant))
				g.Expect(s.Description).ToNot(ContainSubstring("Driver was last alive"))
			}, 50*telem.Millisecond.Duration(), 5*telem.Millisecond.Duration()).Should(Succeed())
		})
	})
})

var _ = Describe("Migration", func() {
	It("Should correctly migrate a v1 rack to a v2 rack", func() {
		db := gorp.Wrap(memkv.New())
		otg := MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
		g := MustSucceed(group.OpenService(ctx, group.Config{DB: db, Ontology: otg}))
		label := MustSucceed(label.OpenService(ctx, label.Config{
			DB:       db,
			Ontology: otg,
			Group:    g,
		}))
		stat := MustSucceed(status.OpenService(ctx, status.ServiceConfig{
			Ontology: otg,
			DB:       db,
			Group:    g,
			Label:    label,
		}))

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
			Status:       stat,
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
