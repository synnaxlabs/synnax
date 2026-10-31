// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack_test

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// stack is a rack service and the services it depends on. Closing it closes all of
// them.
type stack struct {
	xio.MultiCloser
	rack       *rack.Service
	status     *status.Service
	otg        *ontology.Ontology
	driverRole role.Key
}

// openStack opens a rack service and its dependencies on db, as a Core start does. It
// fills every dependency in cfg.
func openStack(ctx context.Context, db *gorp.DB, cfg rack.ServiceConfig) stack {
	GinkgoHelper()
	otg := MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
	g := MustSucceed(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	lbl := MustSucceed(label.OpenService(ctx, label.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    g,
	}))
	stat := MustSucceed(status.OpenService(ctx, status.ServiceConfig{
		Ontology: otg,
		DB:       db,
		Group:    g,
		Label:    lbl,
	}))
	cfg.DB = db
	cfg.Ontology = otg
	cfg.Group = g
	cfg.HostProvider = testutil.NewStaticHostProvider(1)
	cfg.Status = stat
	cfg.DriverRole = MustSucceed(racktestutil.CreateDriverRole(ctx, otg))
	svc := MustSucceed(rack.OpenService(ctx, cfg))
	return stack{
		MultiCloser: xio.MultiCloser{otg, g, lbl, stat, svc},
		rack:        svc,
		status:      stat,
		otg:         otg,
		driverRole:  cfg.DriverRole,
	}
}

// hasDriverRole reports whether the rack with key holds the role driverRole.
func hasDriverRole(
	ctx context.Context,
	otg *ontology.Ontology,
	driverRole role.Key,
	key rack.Key,
) bool {
	GinkgoHelper()
	var children []ontology.Resource
	Expect(otg.NewRetrieve().
		WhereIDs(role.OntologyID(driverRole)).
		TraverseTo(ontology.ChildrenTraverser).
		Entries(&children).
		Exec(ctx, nil)).To(Succeed())
	return slices.ContainsFunc(children, func(r ontology.Resource) bool {
		return r.ID == key.OntologyID()
	})
}

var _ = Describe("Driver role", func() {
	It("Should give a new rack the Driver role", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		s := DeferClose(openStack(ctx, db, rack.ServiceConfig{}))
		r := &rack.Rack{Name: "new"}
		Expect(s.rack.NewWriter(nil).Create(ctx, r)).To(Succeed())
		Expect(hasDriverRole(ctx, s.otg, s.driverRole, r.Key)).To(BeTrue())
	})
	It("Should give the embedded rack the Driver role", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		s := DeferClose(openStack(ctx, db, rack.ServiceConfig{}))
		Expect(hasDriverRole(ctx, s.otg, s.driverRole, s.rack.EmbeddedKey)).
			To(BeTrue())
	})
	It(
		"Should not give the role back when a rack is saved again",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			s := DeferClose(openStack(ctx, db, rack.ServiceConfig{}))
			w := s.rack.NewWriter(nil)
			r := &rack.Rack{Name: "revoked"}
			Expect(w.Create(ctx, r)).To(Succeed())
			Expect(s.otg.NewWriter(nil).DeleteRelationships(ctx, ontology.Relationship{
				From: role.OntologyID(s.driverRole),
				Type: ontology.RelationshipTypeParentOf,
				To:   r.OntologyID(),
			})).To(Succeed())
			Expect(w.Create(ctx, r)).To(Succeed())
			Expect(hasDriverRole(ctx, s.otg, s.driverRole, r.Key)).To(BeFalse())
		},
	)
	It("Should give the Driver role to racks stored before it", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		s := openStack(ctx, db, rack.ServiceConfig{})
		embedded := s.rack.EmbeddedKey
		Expect(s.Close()).To(Succeed())
		Expect(db.Delete(ctx, []byte("gorp.migration.rack.roles"))).To(Succeed())
		// The second open gives the rack a new Driver role, which only the migration
		// can assign to a rack that already exists.
		s = DeferClose(openStack(ctx, db, rack.ServiceConfig{}))
		Expect(hasDriverRole(ctx, s.otg, s.driverRole, embedded)).To(BeTrue())
	})
})

func retrieveStatus(
	ctx context.Context,
	stat *status.Service,
	key rack.Key,
) (rack.Status, error) {
	var s rack.Status
	if err := stat.NewRetrieve[rack.StatusDetails]().
		Where(status.MatchKeys[rack.StatusDetails](rack.StatusKey(key))).
		Entry(&s).
		Exec(ctx, nil); err != nil {
		return rack.Status{}, err
	}
	return s, nil
}

var _ = Describe("Rack", Ordered, func() {
	var (
		writer     rack.Writer
		noTxWriter rack.Writer
		tx         gorp.Tx
		db         *gorp.DB
		svc        *rack.Service
		stat       *status.Service
		otg        *ontology.Ontology
		// frozenNow pins the health monitor's clock to a fixed timestamp when
		// non-zero, letting timing tests stop logical time instead of racing the
		// wall clock. Zero means use the real clock.
		frozenNow atomic.Int64
	)

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New()))
		s := DeferClose(openStack(ctx, db, rack.ServiceConfig{
			HealthCheckInterval: 10 * telem.Millisecond,
			Now: func() telem.TimeStamp {
				if t := frozenNow.Load(); t != 0 {
					return telem.TimeStamp(t)
				}
				return telem.Now()
			},
		}))
		svc, stat, otg = s.rack, s.status, s.otg
		noTxWriter = svc.NewWriter(nil)
	})
	BeforeEach(func(ctx SpecContext) {
		frozenNow.Store(0)
		tx = DeferClose(db.OpenTx())
		writer = svc.NewWriter(tx)
	})
	Describe("Key", func() {
		It(
			"Should correctly construct a key from its components",
			func(ctx SpecContext) {
				k := rack.NewKey(1, 2)
				Expect(k.Node()).To(Equal(node.Key(1)))
				Expect(k).To(Equal(rack.Key(1<<16 | 2)))
			},
		)
	})
	Describe("StatusKey", func() {
		It(
			"Should return the ontology ID string for a rack key",
			func(ctx SpecContext) {
				k := rack.NewKey(1, 2)
				statusKey := rack.StatusKey(k)
				Expect(statusKey).To(Equal(k.OntologyID().String()))
			},
		)
		It(
			"Should return consistent status keys for the same rack key",
			func(ctx SpecContext) {
				k := rack.NewKey(5, 10)
				Expect(rack.StatusKey(k)).To(Equal(rack.StatusKey(k)))
			},
		)
		It(
			"Should return different status keys for different rack keys",
			func(ctx SpecContext) {
				k1 := rack.NewKey(1, 1)
				k2 := rack.NewKey(1, 2)
				Expect(rack.StatusKey(k1)).ToNot(Equal(rack.StatusKey(k2)))
			},
		)
	})
	Describe("Key msgpack decoding", func() {
		codec := msgpack.Codec
		DescribeTable("Should decode rack.Key from various types",
			func(ctx SpecContext, value any, expected rack.Key) {
				data := MustSucceed(codec.Encode(ctx, value))
				var k rack.Key
				Expect(codec.Decode(ctx, data, &k)).To(Succeed())
				Expect(k).To(Equal(expected))
			},
			Entry("string", "65537", rack.Key(65537)),
			Entry("uint64", uint64(65537), rack.Key(65537)),
			Entry("uint32", uint32(65537), rack.Key(65537)),
			Entry("uint16", uint16(1234), rack.Key(1234)),
			Entry("int64", int64(65537), rack.Key(65537)),
			Entry("int32", int32(65537), rack.Key(65537)),
			Entry("float64", float64(65537), rack.Key(65537)),
			Entry("float32", float32(1234), rack.Key(1234)),
		)
		It(
			"Should decode StatusDetails with rack key as float64",
			func(ctx SpecContext) {
				type statusDetailsWithFloat struct {
					Rack float64 `msgpack:"rack"`
				}
				original := statusDetailsWithFloat{
					Rack: float64(65537),
				}
				data := MustSucceed(codec.Encode(ctx, original))
				var decoded rack.StatusDetails
				Expect(codec.Decode(ctx, data, &decoded)).To(Succeed())
				Expect(decoded.Rack).To(Equal(rack.Key(65537)))
			},
		)
		It(
			"Should decode StatusDetails with rack key as string",
			func(ctx SpecContext) {
				type statusDetailsWithString struct {
					Rack string `msgpack:"rack"`
				}
				original := statusDetailsWithString{
					Rack: "65537",
				}
				data := MustSucceed(codec.Encode(ctx, original))
				var decoded rack.StatusDetails
				Expect(codec.Decode(ctx, data, &decoded)).To(Succeed())
				Expect(decoded.Rack).To(Equal(rack.Key(65537)))
			},
		)
	})
	Describe("Create", func() {
		It("Should create a rack and assign it a key", func(ctx SpecContext) {
			r := &rack.Rack{Name: "rack1"}
			Expect(writer.Create(ctx, r)).To(Succeed())
			Expect(!r.Key.IsZero()).To(BeTrue())
			Expect(r.Key).To(Equal(rack.NewKey(1, 2)))
		})
		It("Should correctly increment the local key counter", func(ctx SpecContext) {
			r := &rack.Rack{Name: "rack2"}
			Expect(writer.Create(ctx, r)).To(Succeed())
			Expect(r.Key).To(Equal(rack.NewKey(1, 3)))
		})
		It("Should return an error if the rack has no name", func(ctx SpecContext) {
			r := &rack.Rack{}
			Expect(
				writer.Create(ctx, r),
			).Error().
				To(MatchError(ContainSubstring("name")))
		})
	})
	Describe("CreateMany", func() {
		It("Should create multiple racks", func(ctx SpecContext) {
			racks := []rack.Rack{
				{Name: "rack-many-1"},
				{Name: "rack-many-2"},
			}
			Expect(writer.CreateMany(ctx, &racks)).To(Succeed())

			var retrieved []rack.Rack
			Expect(svc.NewRetrieve().Where(rack.MatchKeys(
				racks[0].Key,
				racks[1].Key,
			)).Entries(&retrieved).Exec(ctx, tx)).To(Succeed())
			Expect(retrieved).To(HaveLen(2))
		})
	})
	Describe("Retrieve", func() {
		It("Should retrieve a rack by its key", func(ctx SpecContext) {
			r := &rack.Rack{Name: "rack3"}
			Expect(writer.Create(ctx, r)).To(Succeed())
			var res rack.Rack
			Expect(
				svc.NewRetrieve().
					Where(rack.MatchKeys(r.Key)).
					Entry(&res).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(res).To(Equal(*r))
		})
		It(
			"Should retrieve racks where the host is the rack's node",
			func(ctx SpecContext) {
				r := &rack.Rack{Name: "rack4"}
				Expect(writer.Create(ctx, r)).To(Succeed())
				var res rack.Rack
				Expect(
					svc.NewRetrieve().
						Where(rack.MatchNodeIsHost(true)).
						Entry(&res).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(res).To(Equal(*r))
			},
		)
		It("Should only retrieve embedded racks", func(ctx SpecContext) {
			var res rack.Rack
			Expect(
				svc.NewRetrieve().
					Where(rack.MatchEmbedded(true)).
					Entry(&res).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(res.Embedded).To(BeTrue())
		})
		Describe("Count", func() {
			It("Should return the number of matching racks", func(ctx SpecContext) {
				initialCount := MustSucceed(svc.NewRetrieve().Count(ctx, tx))
				r1 := &rack.Rack{Name: "count-rack-1"}
				r2 := &rack.Rack{Name: "count-rack-2"}
				Expect(writer.Create(ctx, r1)).To(Succeed())
				Expect(writer.Create(ctx, r2)).To(Succeed())
				newCount := MustSucceed(svc.NewRetrieve().Count(ctx, tx))
				Expect(newCount).To(Equal(initialCount + 2))
			})
			It(
				"Should return the count of racks matching a key filter",
				func(ctx SpecContext) {
					r := &rack.Rack{Name: "count-specific-rack"}
					Expect(writer.Create(ctx, r)).To(Succeed())
					count := MustSucceed(
						svc.NewRetrieve().Where(rack.MatchKeys(r.Key)).Count(ctx, tx),
					)
					Expect(count).To(Equal(1))
				},
			)
		})
		Describe("MatchName", func() {
			It("Should retrieve a rack by its exact name", func(ctx SpecContext) {
				r := &rack.Rack{Name: "unique-rack-name"}
				Expect(writer.Create(ctx, r)).To(Succeed())
				var res rack.Rack
				Expect(
					svc.NewRetrieve().
						Where(rack.MatchNames("unique-rack-name")).
						Entry(&res).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(res.Key).To(Equal(r.Key))
				Expect(res.Name).To(Equal("unique-rack-name"))
			})
			It(
				"Should return not found when name does not match",
				func(ctx SpecContext) {
					r := &rack.Rack{Name: "existing-rack"}
					Expect(writer.Create(ctx, r)).To(Succeed())
					var res rack.Rack
					Expect(svc.NewRetrieve().
						Where(rack.MatchNames("nonexistent-rack")).
						Entry(&res).
						Exec(ctx, tx)).Error().To(MatchError(query.ErrNotFound))
				},
			)
			It("Should filter among multiple racks correctly", func(ctx SpecContext) {
				r1 := &rack.Rack{Name: "filter-rack-alpha"}
				r2 := &rack.Rack{Name: "filter-rack-beta"}
				Expect(writer.Create(ctx, r1)).To(Succeed())
				Expect(writer.Create(ctx, r2)).To(Succeed())
				var res rack.Rack
				Expect(
					svc.NewRetrieve().
						Where(rack.MatchNames("filter-rack-beta")).
						Entry(&res).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(res.Key).To(Equal(r2.Key))
				Expect(res.Name).To(Equal("filter-rack-beta"))
			})
		})
		Describe("MatchIntegration", func() {
			It(
				"Should retrieve a rack that supports the requested integration",
				func(ctx SpecContext) {
					r := &rack.Rack{
						Name:         "ni-rack",
						Integrations: []string{"ni", "opc"},
					}
					Expect(writer.Create(ctx, r)).To(Succeed())
					var res []rack.Rack
					Expect(svc.NewRetrieve().
						Where(rack.MatchIntegration("ni")).
						Entries(&res).
						Exec(ctx, tx)).To(Succeed())
					Expect(res).ToNot(BeEmpty())
					Expect(slices.ContainsFunc(res, func(r rack.Rack) bool {
						return r.Name == "ni-rack"
					})).To(BeTrue())
				},
			)
			It(
				"Should not return racks missing the requested integration",
				func(ctx SpecContext) {
					r := &rack.Rack{
						Name:         "opc-only-rack",
						Integrations: []string{"opc"},
					}
					Expect(writer.Create(ctx, r)).To(Succeed())
					var res []rack.Rack
					Expect(svc.NewRetrieve().
						Where(rack.MatchIntegration("ni")).
						Entries(&res).
						Exec(ctx, tx)).To(Succeed())
					Expect(slices.ContainsFunc(res, func(r rack.Rack) bool {
						return r.Name == "opc-only-rack"
					})).To(BeFalse())
				},
			)
			It(
				"Should not return racks with empty integrations",
				func(ctx SpecContext) {
					r := &rack.Rack{
						Name:         "empty-integrations-rack",
						Integrations: []string{},
					}
					Expect(writer.Create(ctx, r)).To(Succeed())
					var res []rack.Rack
					Expect(svc.NewRetrieve().
						Where(rack.MatchIntegration("ni")).
						Entries(&res).
						Exec(ctx, tx)).To(Succeed())
					Expect(slices.ContainsFunc(res, func(r rack.Rack) bool {
						return r.Name == "empty-integrations-rack"
					})).To(BeFalse())
				},
			)
			It("Should not return racks with nil integrations", func(ctx SpecContext) {
				r := &rack.Rack{Name: "nil-integrations-rack"}
				Expect(writer.Create(ctx, r)).To(Succeed())
				var res []rack.Rack
				Expect(svc.NewRetrieve().
					Where(rack.MatchIntegration("ni")).
					Entries(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(slices.ContainsFunc(res, func(r rack.Rack) bool {
					return r.Name == "nil-integrations-rack"
				})).To(BeFalse())
			})
		})
		Describe("MatchNames", func() {
			It(
				"Should retrieve racks whose name matches any of the provided values",
				func(ctx SpecContext) {
					r1 := &rack.Rack{Name: "names-alpha"}
					r2 := &rack.Rack{Name: "names-beta"}
					r3 := &rack.Rack{Name: "names-gamma"}
					Expect(writer.Create(ctx, r1)).To(Succeed())
					Expect(writer.Create(ctx, r2)).To(Succeed())
					Expect(writer.Create(ctx, r3)).To(Succeed())
					var res []rack.Rack
					Expect(svc.NewRetrieve().
						Where(rack.MatchNames("names-alpha", "names-gamma")).
						Entries(&res).Exec(ctx, tx)).To(Succeed())
					names := lo.Map(
						res,
						func(r rack.Rack, _ int) string { return r.Name },
					)
					Expect(names).To(ContainElements("names-alpha", "names-gamma"))
					Expect(names).ToNot(ContainElement("names-beta"))
				},
			)
		})
		Describe("MatchNode", func() {
			It(
				"Should retrieve racks on a specific cluster node",
				func(ctx SpecContext) {
					r := &rack.Rack{Name: "node-rack"}
					Expect(writer.Create(ctx, r)).To(Succeed())
					var res []rack.Rack
					Expect(svc.NewRetrieve().
						Where(rack.MatchNode(r.Key.Node())).
						Entries(&res).Exec(ctx, tx)).To(Succeed())
					Expect(
						slices.ContainsFunc(
							res,
							func(rr rack.Rack) bool { return rr.Key == r.Key },
						),
					).To(BeTrue())
				},
			)
		})
		Describe("Combinators", func() {
			It("Should compose filters with And", func(ctx SpecContext) {
				hit := &rack.Rack{Name: "combo-and-hit", Integrations: []string{"ni"}}
				miss := &rack.Rack{
					Name:         "combo-and-miss",
					Integrations: []string{"opc"},
				}
				Expect(writer.Create(ctx, hit)).To(Succeed())
				Expect(writer.Create(ctx, miss)).To(Succeed())
				var res []rack.Rack
				Expect(svc.NewRetrieve().Where(rack.And(
					rack.MatchNames("combo-and-hit", "combo-and-miss"),
					rack.MatchIntegration("ni"),
				)).Entries(&res).Exec(ctx, tx)).To(Succeed())
				Expect(res).To(HaveLen(1))
				Expect(res[0].Name).To(Equal("combo-and-hit"))
			})
			It("Should compose filters with Or", func(ctx SpecContext) {
				r1 := &rack.Rack{Name: "combo-or-a", Integrations: []string{"ni"}}
				r2 := &rack.Rack{Name: "combo-or-b", Integrations: []string{"opc"}}
				r3 := &rack.Rack{Name: "combo-or-c", Integrations: []string{"modbus"}}
				Expect(writer.Create(ctx, r1)).To(Succeed())
				Expect(writer.Create(ctx, r2)).To(Succeed())
				Expect(writer.Create(ctx, r3)).To(Succeed())
				var res []rack.Rack
				Expect(svc.NewRetrieve().Where(rack.And(
					rack.MatchNames("combo-or-a", "combo-or-b", "combo-or-c"),
					rack.Or(rack.MatchIntegration("ni"), rack.MatchIntegration("opc")),
				)).Entries(&res).Exec(ctx, tx)).To(Succeed())
				names := lo.Map(res, func(r rack.Rack, _ int) string { return r.Name })
				Expect(names).To(ContainElements("combo-or-a", "combo-or-b"))
				Expect(names).ToNot(ContainElement("combo-or-c"))
			})
			It("Should invert a filter with Not", func(ctx SpecContext) {
				r1 := &rack.Rack{Name: "combo-not-ni", Integrations: []string{"ni"}}
				r2 := &rack.Rack{Name: "combo-not-opc", Integrations: []string{"opc"}}
				Expect(writer.Create(ctx, r1)).To(Succeed())
				Expect(writer.Create(ctx, r2)).To(Succeed())
				var res []rack.Rack
				Expect(svc.NewRetrieve().Where(rack.And(
					rack.MatchNames("combo-not-ni", "combo-not-opc"),
					rack.Not(rack.MatchIntegration("ni")),
				)).Entries(&res).Exec(ctx, tx)).To(Succeed())
				Expect(res).To(HaveLen(1))
				Expect(res[0].Name).To(Equal("combo-not-opc"))
			})
			It(
				"Should compose MatchNodeIsHost under And with MatchNames",
				func(ctx SpecContext) {
					r := &rack.Rack{Name: "combo-host-rack"}
					Expect(writer.Create(ctx, r)).To(Succeed())
					var res []rack.Rack
					Expect(svc.NewRetrieve().Where(rack.And(
						rack.MatchNames("combo-host-rack"),
						rack.MatchNodeIsHost(true),
					)).Entries(&res).Exec(ctx, tx)).To(Succeed())
					Expect(res).To(HaveLen(1))
					Expect(res[0].Key).To(Equal(r.Key))
				},
			)
		})
		Describe("Exists", func() {
			It("Should return true when a rack matches", func(ctx SpecContext) {
				r := &rack.Rack{Name: "exists-rack"}
				Expect(writer.Create(ctx, r)).To(Succeed())
				Expect(MustSucceed(svc.NewRetrieve().
					Where(rack.MatchNames("exists-rack")).
					Exists(ctx, tx))).To(BeTrue())
			})
			It("Should return false when no rack matches", func(ctx SpecContext) {
				Expect(MustSucceed(svc.NewRetrieve().
					Where(rack.MatchNames("does-not-exist-rack")).
					Exists(ctx, tx))).To(BeFalse())
			})
		})
		Describe("Limit and Offset", func() {
			It("Should paginate results with Limit and Offset", func(ctx SpecContext) {
				for i := range 4 {
					Expect(
						writer.Create(
							ctx,
							&rack.Rack{Name: fmt.Sprintf("page-rack-%d", i)},
						),
					).To(Succeed())
				}
				var res []rack.Rack
				Expect(svc.NewRetrieve().
					Where(rack.MatchNames("page-rack-0", "page-rack-1", "page-rack-2", "page-rack-3")).
					Limit(2).
					Offset(1).
					Entries(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res).To(HaveLen(2))
			})
		})
		Describe("Search", func() {
			BeforeEach(func(ctx SpecContext) {
				Expect(writer.Create(ctx, &rack.Rack{Name: "turbopump rack"})).
					To(Succeed())
			})
			It(
				"Should retrieve the racks whose name matches the term",
				func(ctx SpecContext) {
					var res []rack.Rack
					Expect(svc.NewRetrieve().
						Search("turbopump").
						Entries(&res).
						Exec(ctx, tx)).To(Succeed())
					Expect(
						res,
					).To(HaveExactElements(HaveField("Name", "turbopump rack")))
				},
			)
			It("Should count the matches", func(ctx SpecContext) {
				Expect(
					svc.NewRetrieve().Search("turbopump").Count(ctx, tx),
				).To(Equal(1))
			})
			It("Should report whether a match exists", func(ctx SpecContext) {
				Expect(
					svc.NewRetrieve().Search("turbopump").Exists(ctx, tx),
				).To(BeTrue())
				Expect(svc.NewRetrieve().Search("valve").Exists(ctx, tx)).To(BeFalse())
			})
			It("Should add no filter for an empty term", func(ctx SpecContext) {
				Expect(svc.NewRetrieve().Search("").Count(ctx, tx)).
					To(Equal(MustSucceed(svc.NewRetrieve().Count(ctx, tx))))
			})
		})
	})
	Describe("Search", func() {
		create := func(ctx SpecContext, name string) ontology.ID {
			GinkgoHelper()
			r := rack.Rack{Name: name}
			Expect(writer.Create(ctx, &r)).To(Succeed())
			return r.OntologyID()
		}

		It(
			"Should return the matching racks, closest match first",
			func(ctx SpecContext) {
				a := create(ctx, "turbopump test")
				b := create(ctx, "turbopump")
				Expect(MustSucceed(svc.Search(ctx, tx, "turbopump", 0)).Keys).
					To(Equal([]ontology.ID{b, a}))
				Expect(MustSucceed(svc.Search(ctx, tx, "turbopump", 1)).Keys).
					To(Equal([]ontology.ID{b}))
			},
		)

		It(
			"Should not match a staged rack outside its transaction",
			func(ctx SpecContext) {
				create(ctx, "injector")
				Expect(
					MustSucceed(svc.Search(ctx, db, "injector", 0)).Keys,
				).To(BeEmpty())
			},
		)
	})

	Describe("Create with Integrations", func() {
		It("Should persist integrations on the rack", func(ctx SpecContext) {
			r := &rack.Rack{
				Name:         "persist-integ-rack",
				Integrations: []string{"ni", "opc", "arc"},
			}
			Expect(writer.Create(ctx, r)).To(Succeed())
			var res rack.Rack
			Expect(
				svc.NewRetrieve().
					Where(rack.MatchKeys(r.Key)).
					Entry(&res).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(res.Integrations).To(ConsistOf([]string{"ni", "opc", "arc"}))
		})
		It("Should update integrations on upsert", func(ctx SpecContext) {
			r := &rack.Rack{Name: "upsert-integ-rack", Integrations: []string{"ni"}}
			Expect(writer.Create(ctx, r)).To(Succeed())
			r.Integrations = []string{"opc", "modbus"}
			Expect(writer.Create(ctx, r)).To(Succeed())
			var res rack.Rack
			Expect(
				svc.NewRetrieve().
					Where(rack.MatchKeys(r.Key)).
					Entry(&res).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(res.Integrations).To(ConsistOf([]string{"opc", "modbus"}))
		})
	})
	Describe("Rename", func() {
		It(
			"Should rename the embedded rack and its status, keeping every other field",
			func(ctx SpecContext) {
				var before rack.Rack
				Expect(svc.NewRetrieve().
					Where(rack.MatchKeys(svc.EmbeddedKey)).
					Entry(&before).
					Exec(ctx, tx)).To(Succeed())
				Expect(writer.Rename(ctx, svc.EmbeddedKey, "Renamed Embedded")).
					To(Succeed())
				var after rack.Rack
				Expect(svc.NewRetrieve().
					Where(rack.MatchKeys(svc.EmbeddedKey)).
					Entry(&after).
					Exec(ctx, tx)).To(Succeed())
				expected := before
				expected.Name = "Renamed Embedded"
				Expect(after).To(Equal(expected))
				Expect(after.Embedded).To(BeTrue())
				var renamed rack.Status
				Expect(stat.NewRetrieve[rack.StatusDetails]().
					Where(status.MatchKeys[rack.StatusDetails](
						rack.StatusKey(svc.EmbeddedKey),
					)).
					Entry(&renamed).
					Exec(ctx, tx)).To(Succeed())
				Expect(renamed.Name).To(Equal("Renamed Embedded"))
				Expect(renamed.Details.Rack).To(Equal(svc.EmbeddedKey))
			},
		)
		It("Should keep a rack's integrations", func(ctx SpecContext) {
			r := &rack.Rack{Name: "rename-integ-rack", Integrations: []string{"ni"}}
			Expect(writer.Create(ctx, r)).To(Succeed())
			Expect(writer.Rename(ctx, r.Key, "renamed-integ-rack")).To(Succeed())
			var res rack.Rack
			Expect(svc.NewRetrieve().
				Where(rack.MatchKeys(r.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("renamed-integ-rack"))
			Expect(res.Integrations).To(Equal([]string{"ni"}))
		})
		It("Should return a validation error for an empty name", func(ctx SpecContext) {
			r := &rack.Rack{Name: "rename-empty-rack"}
			Expect(writer.Create(ctx, r)).To(Succeed())
			Expect(writer.Rename(ctx, r.Key, "")).
				To(MatchError(ContainSubstring("name: required")))
		})
		It("Should return a not found error for a missing rack", func(ctx SpecContext) {
			Expect(writer.Rename(ctx, rack.NewKey(1, 60000), "renamed")).
				To(MatchError(query.ErrNotFound))
		})
		It("Should notify observers of the rename", func(ctx SpecContext) {
			r := &rack.Rack{Name: "observed-rack"}
			Expect(noTxWriter.Create(ctx, r)).To(Succeed())
			renameTx := db.OpenTx()
			defer func() { Expect(renameTx.Close()).To(Succeed()) }()
			Expect(svc.NewWriter(renameTx).Rename(ctx, r.Key, "renamed")).
				To(Succeed())
			var names []string
			disconnect := svc.Observe().OnChange(func(
				_ context.Context,
				reader gorp.TxReader[rack.Key, rack.Rack],
			) {
				for c := range reader {
					names = append(names, c.Value.Name)
				}
			})
			defer disconnect()
			Expect(renameTx.Commit(ctx)).To(Succeed())
			Expect(names).To(Equal([]string{"renamed"}))
		})
	})
	Describe("Delete", func() {
		It("Should delete a rack and its associated status", func(ctx SpecContext) {
			r := &rack.Rack{Name: "rack4"}
			Expect(writer.Create(ctx, r)).To(Succeed())
			Expect(writer.DeleteGuard(ctx, r.Key, nil)).To(Succeed())
			var res rack.Rack
			Expect(
				svc.NewRetrieve().
					Where(rack.MatchKeys(r.Key)).
					Entry(&res).
					Exec(ctx, tx),
			).To(MatchError(query.ErrNotFound))
			var deletedStatus rack.Status
			Expect(stat.NewRetrieve[rack.StatusDetails]().
				Where(status.MatchKeys[rack.StatusDetails](r.OntologyID().String())).
				Entry(&deletedStatus).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})

		It("Should delete the rack's ontology resource", func(ctx SpecContext) {
			r := &rack.Rack{Name: "rack5"}
			Expect(writer.Create(ctx, r)).To(Succeed())
			Expect(otg.NewRetrieve().
				WhereIDs(r.OntologyID()).
				Entries(&[]ontology.Resource{}).
				Exec(ctx, tx)).To(Succeed())
			Expect(writer.DeleteGuard(ctx, r.Key, nil)).To(Succeed())
			Expect(otg.NewRetrieve().
				WhereIDs(r.OntologyID()).
				Entries(&[]ontology.Resource{}).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Embedded Rack", func() {
		It("Should correctly create the node embedded rack", func(ctx SpecContext) {
			Expect(svc.EmbeddedKey).ToNot(Equal(rack.Key(0)))
			var embeddedRack rack.Rack
			Expect(
				svc.NewRetrieve().
					Where(rack.MatchKeys(svc.EmbeddedKey)).
					Entry(&embeddedRack).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(embeddedRack.Embedded).To(BeTrue())
		})
		It(
			"Should keep a renamed embedded rack across restarts",
			func(ctx SpecContext) {
				restartDB := DeferClose(gorp.Wrap(memkv.New()))
				before := openStack(ctx, restartDB, rack.ServiceConfig{})
				key := before.rack.EmbeddedKey
				Expect(
					before.rack.NewWriter(nil).Rename(ctx, key, "Renamed"),
				).To(Succeed())
				Expect(before.Close()).To(Succeed())
				after := DeferClose(openStack(ctx, restartDB, rack.ServiceConfig{}))
				Expect(after.rack.EmbeddedKey).To(Equal(key))
				var embedded []rack.Rack
				Expect(after.rack.NewRetrieve().
					Where(rack.MatchEmbedded(true)).
					Entries(&embedded).
					Exec(ctx, nil)).To(Succeed())
				Expect(embedded).To(HaveLen(1))
				Expect(embedded[0].Name).To(Equal("Renamed"))
			},
		)
		It(
			"Should find an embedded rack stored without a key binding by its name",
			func(ctx SpecContext) {
				restartDB := DeferClose(gorp.Wrap(memkv.New()))
				before := openStack(ctx, restartDB, rack.ServiceConfig{})
				key := before.rack.EmbeddedKey
				Expect(before.Close()).To(Succeed())
				Expect(restartDB.Delete(
					ctx,
					[]byte("1.rack.embedded.Node 1 Embedded Driver"),
				)).To(Succeed())
				after := DeferClose(openStack(ctx, restartDB, rack.ServiceConfig{}))
				Expect(after.rack.EmbeddedKey).To(Equal(key))
			},
		)
		It(
			"Should apply edit to the embedded rack before saving it",
			func(ctx SpecContext) {
				r := MustSucceed(svc.RetrieveOrCreateEmbedded(
					ctx,
					"edited embedded",
					func(r *rack.Rack) { r.Integrations = []string{"arc"} },
				))
				var res rack.Rack
				Expect(svc.NewRetrieve().
					Where(rack.MatchKeys(r.Key)).
					Entry(&res).
					Exec(ctx, nil)).To(Succeed())
				Expect(res.Name).To(Equal("edited embedded"))
				Expect(res.Embedded).To(BeTrue())
				Expect(res.Integrations).To(Equal([]string{"arc"}))
			},
		)
		It(
			"Should return a zero rack when the embedded rack fails to save",
			func(ctx SpecContext) {
				r, err := svc.RetrieveOrCreateEmbedded(
					ctx,
					"invalid embedded",
					func(r *rack.Rack) { r.Name = "" },
				)
				Expect(err).To(MatchError(ContainSubstring("name: required")))
				Expect(r).To(Equal(rack.Rack{}))
			},
		)
	})

	Describe("Status", func() {
		It("Should initialize a rack with an unknown status", func(ctx SpecContext) {
			r := rack.Rack{Name: "test rack"}
			Expect(noTxWriter.Create(ctx, &r)).To(Succeed())
			s := MustSucceed(retrieveStatus(ctx, stat, r.Key))
			Expect(s.Message).To(Equal("Rack status unknown"))
			Expect(s.Variant).To(Equal(status.VariantWarning))
			Expect(s.Time).To(BeNumerically("~", telem.Now(), 3*telem.SecondTS))
			Expect(s.Key).To(ContainSubstring(string(ontology.ResourceTypeRack)))
			Expect(s.Details.Rack).To(Equal(r.Key))
		})

		It(
			"Should use the provided status when creating a rack",
			func(ctx SpecContext) {
				providedStatus := &rack.Status{
					Variant:     status.VariantSuccess,
					Time:        telem.Now(),
					Message:     "Custom status message",
					Description: "Custom description",
				}
				r := rack.Rack{Name: "rack with custom status", Status: providedStatus}
				Expect(noTxWriter.Create(ctx, &r)).To(Succeed())
				s := MustSucceed(retrieveStatus(ctx, stat, r.Key))
				Expect(s.Message).To(Equal("Custom status message"))
				Expect(s.Description).To(Equal("Custom description"))
				Expect(s.Variant).To(Equal(status.VariantSuccess))
				// Key should be auto-assigned to match ontology ID
				Expect(s.Key).To(Equal(r.OntologyID().String()))
				// Time should be auto-filled
				Expect(s.Time).To(BeNumerically("~", telem.Now(), 3*telem.SecondTS))
				// Name should be auto-filled from rack name
				Expect(s.Name).To(Equal(r.Name))
				// Details.Rack should be auto-filled
				Expect(s.Details.Rack).To(Equal(r.Key))
			},
		)

		It(
			"Should return a validation error if provided status has empty variant",
			func(ctx SpecContext) {
				providedStatus := &rack.Status{
					Message: "Status with no variant",
					Time:    telem.Now(),
				}
				r := rack.Rack{Name: "rack with invalid status", Status: providedStatus}
				Expect(
					noTxWriter.Create(ctx, &r),
				).Error().
					To(MatchError(ContainSubstring("variant")))
			},
		)

		It(
			"Should restore a missing status row when the rack is re-configured",
			func(ctx SpecContext) {
				r := rack.Rack{Name: "heal rack"}
				Expect(writer.Create(ctx, &r)).To(Succeed())

				Expect(stat.NewWriter(tx).
					Delete(ctx, r.OntologyID().String())).To(Succeed())
				Expect(stat.NewRetrieve[rack.StatusDetails]().
					Where(status.MatchKeys[rack.StatusDetails](r.OntologyID().String())).
					Exec(ctx, tx)).To(MatchError(query.ErrNotFound))

				reconfigured := rack.Rack{Key: r.Key, Name: r.Name}
				Expect(writer.Create(ctx, &reconfigured)).To(Succeed())

				var healed rack.Status
				Expect(stat.NewRetrieve[rack.StatusDetails]().
					Where(status.MatchKeys[rack.StatusDetails](r.OntologyID().String())).
					Entry(&healed).
					Exec(ctx, tx)).To(Succeed())
				Expect(healed.Details.Rack).To(Equal(r.Key))
			},
		)

		It(
			"Should not clobber a live status row on a no-op re-configure",
			func(ctx SpecContext) {
				r := rack.Rack{
					Name: "live status rack",
					Status: &rack.Status{
						Variant: status.VariantSuccess,
						Message: "Rack is connected",
						Time:    telem.Now(),
					},
				}
				Expect(writer.Create(ctx, &r)).To(Succeed())

				reconfigured := rack.Rack{Key: r.Key, Name: r.Name}
				Expect(writer.Create(ctx, &reconfigured)).To(Succeed())

				var preserved rack.Status
				Expect(stat.NewRetrieve[rack.StatusDetails]().
					Where(status.MatchKeys[rack.StatusDetails](r.OntologyID().String())).
					Entry(&preserved).
					Exec(ctx, tx)).To(Succeed())
				Expect(preserved.Variant).To(Equal(status.VariantSuccess))
				Expect(preserved.Message).To(Equal("Rack is connected"))
			},
		)

		It(
			"Should mark a rack as dead when it doesn't receive a status within the health check interval",
			func(ctx SpecContext) {
				r := rack.Rack{Name: "dead test rack"}
				Expect(noTxWriter.Create(ctx, &r)).To(Succeed())

				Eventually(func(g Gomega) {
					s := MustSucceed(retrieveStatus(ctx, stat, r.Key))
					g.Expect(s.Message).To(Equal("Driver not running"))
					g.Expect(s.Variant).To(Equal(status.VariantWarning))
					g.Expect(s.Time).
						To(BeNumerically("~", telem.Now(), 3*telem.SecondTS))
					g.Expect(s.Key).
						To(ContainSubstring(string(ontology.ResourceTypeRack)))
					g.Expect(s.Details.Rack).To(Equal(r.Key))
					g.Expect(s.Description).To(Equal(
						"The Core has not heard from the Driver on dead test rack. " +
							"Check that the Driver is running and can reach the Core.",
					))
				}).Should(Succeed())
			},
		)

		It(
			"Should not mark a rack as dead when the status is actively updated",
			func(ctx SpecContext) {
				// Freeze logical time so that, from the monitor's perspective, no time
				// elapses after a rack reports healthy. The health check ticker keeps
				// firing on the real clock, but now - lastUpdated stays zero, so an
				// actively-updated rack must never be marked dead.
				frozenNow.Store(int64(telem.Now()))

				r := rack.Rack{Name: "active test rack"}
				Expect(noTxWriter.Create(ctx, &r)).To(Succeed())

				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return stat.NewWriter(tx).Set(ctx, &rack.Status{
						Key:     r.OntologyID().String(),
						Name:    r.Name,
						Time:    telem.Now(),
						Variant: status.VariantSuccess,
						Message: "Running",
						Details: rack.StatusDetails{Rack: r.Key},
					})
				})).To(Succeed())

				Consistently(func(g Gomega) {
					s := MustSucceed(retrieveStatus(ctx, stat, r.Key))
					g.Expect(s.Message).To(Equal("Running"))
					g.Expect(s.Variant).To(Equal(status.VariantSuccess))
					g.Expect(s.Description).To(BeEmpty())
				}, 50*telem.Millisecond.Duration(), 5*telem.Millisecond.Duration()).Should(Succeed())
			},
		)

		It("Should write each outage once", func(ctx SpecContext) {
			const name = "outage test rack"
			var writes, alerts atomic.Int32
			DeferCleanup(stat.Observe().OnChange(func(
				_ context.Context,
				t gorp.TxReader[status.Key, status.Status[any]],
			) {
				for ch := range t {
					if ch.Value.Name == name &&
						ch.Value.Message == "Driver not running" {
						writes.Add(1)
					}
				}
			}))
			DeferCleanup(svc.OnSuspect(func(_ context.Context, s rack.Status) error {
				if s.Name == name {
					alerts.Add(1)
				}
				return nil
			}))
			r := rack.Rack{Name: name}
			Expect(noTxWriter.Create(ctx, &r)).To(Succeed())

			Eventually(alerts.Load).Should(BeEquivalentTo(1))
			Consistently(func(g Gomega) {
				g.Expect(writes.Load()).To(BeEquivalentTo(1))
				g.Expect(alerts.Load()).To(BeEquivalentTo(1))
			}, 300*time.Millisecond, 10*time.Millisecond).Should(Succeed())
		})

		It(
			"Should copy an outage again on the next check after a failed copy",
			func(ctx SpecContext) {
				const name = "copy retry test rack"
				var failing, succeeding atomic.Int32
				DeferCleanup(
					svc.OnSuspect(func(_ context.Context, s rack.Status) error {
						if s.Name == name && failing.Add(1) == 1 {
							return errors.New("copy failed")
						}
						return nil
					}),
				)
				DeferCleanup(
					svc.OnSuspect(func(_ context.Context, s rack.Status) error {
						if s.Name == name {
							succeeding.Add(1)
						}
						return nil
					}),
				)
				r := rack.Rack{Name: name}
				Expect(noTxWriter.Create(ctx, &r)).To(Succeed())

				Eventually(failing.Load).Should(BeEquivalentTo(2))
				Consistently(func(g Gomega) {
					g.Expect(failing.Load()).To(BeEquivalentTo(2))
					g.Expect(succeeding.Load()).To(BeEquivalentTo(1))
				}, 300*time.Millisecond, 10*time.Millisecond).Should(Succeed())
			},
		)

		It(
			"Should write a new outage after the rack reports healthy",
			func(ctx SpecContext) {
				r := rack.Rack{Name: "reset test rack"}
				Expect(noTxWriter.Create(ctx, &r)).To(Succeed())

				var (
					alertCount int
					alertMu    sync.Mutex
				)
				getCount := func() int {
					alertMu.Lock()
					defer alertMu.Unlock()
					return alertCount
				}
				disconnect := svc.OnSuspect(
					func(_ context.Context, s rack.Status) error {
						if s.Details.Rack == r.Key {
							alertMu.Lock()
							alertCount++
							alertMu.Unlock()
						}
						return nil
					},
				)
				defer disconnect()

				Eventually(getCount).Should(Equal(1))

				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return stat.NewWriter(tx).Set(ctx, &rack.Status{
						Key:     r.OntologyID().String(),
						Name:    r.Name,
						Time:    telem.Now(),
						Variant: status.VariantSuccess,
						Message: "Running",
						Details: rack.StatusDetails{Rack: r.Key},
					})
				})).To(Succeed())

				countAfterRecovery := getCount()
				Eventually(
					getCount,
					50*time.Millisecond,
					5*time.Millisecond,
				).Should(BeNumerically(">", countAfterRecovery))
			},
		)

		It(
			"Should mark a stored healthy rack as dead when it is silent after a restart",
			func(ctx SpecContext) {
				restartDB := DeferClose(gorp.Wrap(memkv.New()))
				before := openStack(ctx, restartDB, rack.ServiceConfig{
					HealthCheckInterval: telem.Hour,
				})
				r := rack.Rack{
					Name: "restarted rack",
					Status: &rack.Status{
						Variant: status.VariantSuccess,
						Message: "Driver is running",
						Time:    telem.Now(),
					},
				}
				Expect(before.rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
				Expect(before.Close()).To(Succeed())

				after := DeferClose(openStack(ctx, restartDB, rack.ServiceConfig{
					HealthCheckInterval: 10 * telem.Millisecond,
				}))
				Eventually(func(g Gomega) {
					s := MustSucceed(retrieveStatus(ctx, after.status, r.Key))
					g.Expect(s.Variant).To(Equal(status.VariantWarning))
					g.Expect(s.Message).To(Equal("Driver not running"))
				}).Should(Succeed())
			},
		)
	})
})
