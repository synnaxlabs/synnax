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
	"bytes"
	"context"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

// stack is a rack service and the services it depends on. Closing it closes all of
// them.
type stack struct {
	xio.MultiCloser
	rack       *rack.Service
	status     *status.Service
	group      *group.Service
	driverRole role.Key
}

// openStack opens a rack service and its dependencies on db, as a Core start does. It
// fills every dependency in cfg, and creates a Driver role when cfg has none.
func openStack(ctx context.Context, db *gorp.DB, cfg rack.ServiceConfig) stack {
	GinkgoHelper()
	g := MustSucceed(group.OpenService(ctx, group.ServiceConfig{
		DB: db,
	}))
	lbl := MustSucceed(label.OpenService(ctx, label.ServiceConfig{
		DB: db,
	}))
	stat := MustSucceed(status.OpenService(ctx, status.ServiceConfig{
		DB: db,
	}))
	cfg.DB = db
	cfg.HostKey = 1
	cfg.Status = stat
	roles := MustSucceed(role.OpenService(ctx, role.ServiceConfig{
		DB: db,
	}))
	if cfg.DriverRole == uuid.Nil() {
		cfg.DriverRole = MustSucceed(racktestutil.CreateDriverRole(ctx, db, roles))
	}
	svc := MustSucceed(rack.OpenService(ctx, cfg))
	return stack{
		MultiCloser: xio.MultiCloser{g, lbl, stat, roles, svc},
		rack:        svc,
		status:      stat,
		group:       g,
		driverRole:  cfg.DriverRole,
	}
}

// hasDriverRole reports whether the rack with key holds the stack's Driver role.
func hasDriverRole(ctx context.Context, s stack, key rack.Key) bool {
	GinkgoHelper()
	var r rack.Rack
	Expect(s.rack.NewRetrieve().
		Where(rack.MatchKeys(key)).
		Entry(&r).
		Exec(ctx, s.rack.DB)).To(Succeed())
	return r.Role == s.driverRole
}

var _ = Describe("Driver role", func() {
	It("Should give a new rack the Driver role", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		s := DeferClose(openStack(ctx, db, rack.ServiceConfig{}))
		r := &rack.Rack{Name: "new"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return s.rack.NewWriter(tx).Create(ctx, r)
		})).To(Succeed())
		Expect(hasDriverRole(ctx, s, r.Key)).To(BeTrue())
	})
	It("Should give the embedded rack the Driver role", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		s := DeferClose(openStack(ctx, db, rack.ServiceConfig{}))
		Expect(hasDriverRole(ctx, s, s.rack.EmbeddedKey)).
			To(BeTrue())
	})
	It("Should give the Driver role to racks stored before it", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		s := openStack(ctx, db, rack.ServiceConfig{})
		driver := s.driverRole
		Expect(s.Close()).To(Succeed())
		stored := rack.Rack{Key: rack.NewKey(1, 10), Name: "stored"}
		Expect(gorp.WrapWriter[rack.Key, rack.Rack](db).Set(ctx, stored)).
			To(Succeed())
		Expect(db.Delete(ctx, []byte("gorp.migration.rack.roles"))).To(Succeed())
		s = DeferClose(openStack(ctx, db, rack.ServiceConfig{DriverRole: driver}))
		Expect(hasDriverRole(ctx, s, stored.Key)).To(BeTrue())
	})
})

// failingDB rejects each commit that fail returns an error for. fail receives the
// values that the commit sets.
type failingDB struct {
	kv.DB
	fail func(values [][]byte) error
}

func (d failingDB) OpenTx() kv.Tx { return &failingTx{Tx: d.DB.OpenTx(), fail: d.fail} }

type failingTx struct {
	kv.Tx
	fail   func(values [][]byte) error
	values [][]byte
}

func (t *failingTx) Set(ctx context.Context, key, value []byte) error {
	t.values = append(t.values, slices.Clone(value))
	return t.Tx.Set(ctx, key, value)
}

func (t *failingTx) Commit(ctx context.Context, opts ...any) error {
	if err := t.fail(t.values); err != nil {
		return err
	}
	return t.Tx.Commit(ctx, opts...)
}

func retrieveStatusTx(
	ctx context.Context,
	stat *status.Service,
	tx gorp.Tx,
	key rack.Key,
) (rack.Status, error) {
	var s rack.Status
	if err := stat.NewRetrieve[rack.StatusDetails]().
		Where(status.MatchOwners[rack.StatusDetails](key.ResourceID())).
		Entry(&s).
		Exec(ctx, tx); err != nil {
		return rack.Status{}, err
	}
	return s, nil
}

var _ = Describe("Rack", Ordered, func() {
	var (
		writer rack.Writer
		tx     gorp.Tx
		db     *gorp.DB
		svc    *rack.Service
		stat   *status.Service
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
		svc, stat = s.rack, s.status
	})
	BeforeEach(func(ctx SpecContext) { frozenNow.Store(0) })
	openTx := func() {
		tx = DeferClose(db.OpenTx())
		writer = svc.NewWriter(tx)
	}
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
	Describe("Key msgpack decoding", func() {
		codec := msgpack.Codec
		DescribeTable("Should decode rack.Key from various types",
			func(ctx SpecContext, value any, expected rack.Key) {
				data := MustSucceed(codec.Encode(value))
				var k rack.Key
				Expect(codec.Decode(data, &k)).To(Succeed())
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
		type details struct {
			Rack rack.Key `msgpack:"rack"`
		}
		It(
			"Should decode a struct with a rack key as float64",
			func(ctx SpecContext) {
				type statusDetailsWithFloat struct {
					Rack float64 `msgpack:"rack"`
				}
				original := statusDetailsWithFloat{
					Rack: float64(65537),
				}
				data := MustSucceed(codec.Encode(original))
				var decoded details
				Expect(codec.Decode(data, &decoded)).To(Succeed())
				Expect(decoded.Rack).To(Equal(rack.Key(65537)))
			},
		)
		It(
			"Should decode a struct with a rack key as string",
			func(ctx SpecContext) {
				type statusDetailsWithString struct {
					Rack string `msgpack:"rack"`
				}
				original := statusDetailsWithString{
					Rack: "65537",
				}
				data := MustSucceed(codec.Encode(original))
				var decoded details
				Expect(codec.Decode(data, &decoded)).To(Succeed())
				Expect(decoded.Rack).To(Equal(rack.Key(65537)))
			},
		)
	})
	Describe("Create", func() {
		BeforeEach(openTx)
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
		It("Should refuse a rack whose key exists", func(ctx SpecContext) {
			r := rack.Rack{Name: "original", Integrations: []string{"ni"}}
			Expect(writer.Create(ctx, &r)).To(Succeed())
			again := rack.Rack{Key: r.Key, Name: "again"}
			Expect(writer.Create(ctx, &again)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("rack %s already exists", r.Key)),
			))
			var res rack.Rack
			Expect(svc.NewRetrieve().
				Where(rack.MatchKeys(r.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("original"))
			Expect(res.Integrations).To(Equal([]string{"ni"}))
		})
	})
	Describe("CreateMany", func() {
		BeforeEach(openTx)
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
		BeforeEach(openTx)
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
			expected := *r
			expected.Status = nil
			Expect(res).To(Equal(expected))
		})
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
		Describe("Limit and After", func() {
			It("Should page through results", func(ctx SpecContext) {
				names := make([]string, 4)
				created := make([]rack.Key, 4)
				for i := range names {
					names[i] = fmt.Sprintf("page-rack-%d", i)
					r := &rack.Rack{Name: names[i]}
					Expect(writer.Create(ctx, r)).To(Succeed())
					created[i] = r.Key
				}
				page := func(q rack.Retrieve) []rack.Key {
					var res []rack.Rack
					Expect(q.Where(rack.MatchNames(names...)).Limit(3).Entries(&res).
						Exec(ctx, tx)).To(Succeed())
					return lo.Map(
						res,
						func(r rack.Rack, _ int) rack.Key { return r.Key },
					)
				}
				first := page(svc.NewRetrieve())
				second := page(svc.NewRetrieve().After(first[2]))
				Expect(second).To(HaveLen(1))
				Expect(slices.Concat(first, second)).To(ConsistOf(created))
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
		BeforeEach(openTx)
		create := func(ctx SpecContext, name string) resource.ID {
			GinkgoHelper()
			r := rack.Rack{Name: name}
			Expect(writer.Create(ctx, &r)).To(Succeed())
			return r.ResourceID()
		}

		It(
			"Should return the matching racks, closest match first",
			func(ctx SpecContext) {
				a := create(ctx, "turbopump test")
				b := create(ctx, "turbopump")
				q := search.Query[resource.ID]{Term: "turbopump"}
				Expect(MustSucceed(svc.Resources().Search(ctx, tx, q)).Keys).
					To(Equal([]resource.ID{b, a}))
				q.Limit = 1
				Expect(MustSucceed(svc.Resources().Search(ctx, tx, q)).Keys).
					To(Equal([]resource.ID{b}))
			},
		)

		It(
			"Should not match a staged rack outside its transaction",
			func(ctx SpecContext) {
				create(ctx, "injector")
				q := search.Query[resource.ID]{Term: "injector"}
				Expect(
					MustSucceed(svc.Resources().Search(ctx, db, q)).Keys,
				).To(BeEmpty())
			},
		)
	})

	Describe("Resources", func() {
		BeforeEach(openTx)
		It("Should return the ID and name of each rack", func(ctx SpecContext) {
			a := rack.Rack{Name: "rack a"}
			b := rack.Rack{Name: "rack b"}
			Expect(writer.Create(ctx, &a)).To(Succeed())
			Expect(writer.Create(ctx, &b)).To(Succeed())
			Expect(svc.Resources().Retrieve(
				ctx,
				tx,
				[]resource.ID{a.ResourceID(), b.ResourceID()},
			)).To(ConsistOf(
				resource.Resource{ID: a.ResourceID(), Name: "rack a"},
				resource.Resource{ID: b.ResourceID(), Name: "rack b"},
			))
		})

		It("Should return a validation error for a key that is not a rack key",
			func(ctx SpecContext) {
				Expect(svc.Resources().Retrieve(
					ctx,
					tx,
					[]resource.ID{{Type: resource.TypeRack, Key: "pump"}},
				)).Error().To(And(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(`invalid rack key "pump"`)),
				))
			},
		)

		It("Should leave out an ID that names no rack", func(ctx SpecContext) {
			r := rack.Rack{Name: "present rack"}
			Expect(writer.Create(ctx, &r)).To(Succeed())
			Expect(svc.Resources().Retrieve(
				ctx,
				tx,
				[]resource.ID{rack.Key(4294967295).ResourceID(), r.ResourceID()},
			)).To(Equal([]resource.Resource{{ID: r.ResourceID(), Name: r.Name}}))
		})

		It("Should list each rack by its name", func(ctx SpecContext) {
			r := rack.Rack{Name: "listed rack"}
			Expect(writer.Create(ctx, &r)).To(Succeed())
			Expect(svc.Resources().List(ctx, tx, 10000, nil)).To(ContainElement(
				resource.Resource{ID: r.ResourceID(), Name: r.Name},
			))
		})
	})

	Describe("Create with Integrations", func() {
		BeforeEach(openTx)
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
	})
	Describe("Rename", func() {
		It(
			"Should rename the embedded rack and its status, keeping every other field",
			func(ctx SpecContext) {
				openTx()
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
				// A path action stores a rack with no integrations as an empty list.
				expected.Integrations = []string{}
				Expect(after).To(Equal(expected))
				Expect(after.Embedded).To(BeTrue())
				renamed := MustSucceed(retrieveStatusTx(ctx, stat, tx, svc.EmbeddedKey))
				Expect(renamed.Name).To(Equal("Renamed Embedded"))
				Expect(renamed.Owner).To(HaveValue(Equal(svc.EmbeddedKey.ResourceID())))
			},
		)
		It("Should keep a rack's integrations", func(ctx SpecContext) {
			openTx()
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
		It("Should write the default status when the rack has none", func(
			ctx SpecContext,
		) {
			openTx()
			r := &rack.Rack{Name: "rename-no-status-rack"}
			Expect(writer.Create(ctx, r)).To(Succeed())
			Expect(stat.NewWriter(tx).DeleteOwned(ctx, r.ResourceID())).To(Succeed())
			Expect(writer.Rename(ctx, r.Key, "renamed-no-status-rack")).To(Succeed())
			s := MustSucceed(retrieveStatusTx(ctx, stat, tx, r.Key))
			Expect(s.Name).To(Equal("renamed-no-status-rack"))
			Expect(s.Variant).To(Equal(status.VariantWarning))
			Expect(s.Message).To(Equal("Rack status unknown"))
		})
		It("Should return a validation error for an empty name", func(ctx SpecContext) {
			openTx()
			r := &rack.Rack{Name: "rename-empty-rack"}
			Expect(writer.Create(ctx, r)).To(Succeed())
			Expect(writer.Rename(ctx, r.Key, "")).
				To(MatchError(ContainSubstring("name: required")))
		})
		It("Should return a not found error for a missing rack", func(ctx SpecContext) {
			openTx()
			Expect(writer.Rename(ctx, rack.NewKey(1, 60000), "renamed")).
				To(MatchError(query.ErrNotFound))
		})
		It(
			"Should return the rename in the changes of its commit",
			func(ctx SpecContext) {
				r := &rack.Rack{Name: "observed-rack"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, r)
				})).To(Succeed())
				renameTx := db.OpenTx()
				defer func() { Expect(renameTx.Close()).To(Succeed()) }()
				Expect(svc.NewWriter(renameTx).Rename(ctx, r.Key, "renamed")).
					To(Succeed())
				var names []string
				disconnect := db.Observe().
					OnChange(func(ctx context.Context, c gorp.Commit) {
						for ch := range svc.Changes(ctx, c) {
							names = append(names, ch.Value.Name)
						}
					})
				defer disconnect()
				Expect(renameTx.Commit(ctx)).To(Succeed())
				Expect(names).To(Equal([]string{"renamed"}))
			},
		)
	})
	Describe("Delete", func() {
		BeforeEach(openTx)
		It("Should delete a rack and its associated status", func(ctx SpecContext) {
			r := &rack.Rack{Name: "rack4"}
			Expect(writer.Create(ctx, r)).To(Succeed())
			Expect(writer.Delete(ctx, nil, r.Key)).To(Succeed())
			var res rack.Rack
			Expect(
				svc.NewRetrieve().
					Where(rack.MatchKeys(r.Key)).
					Entry(&res).
					Exec(ctx, tx),
			).To(MatchError(query.ErrNotFound))
			Expect(retrieveStatusTx(ctx, stat, tx, r.Key)).Error().
				To(MatchError(query.ErrNotFound))
		})

		It("Should delete many racks at once", func(ctx SpecContext) {
			a := &rack.Rack{Name: "a"}
			b := &rack.Rack{Name: "b"}
			Expect(writer.Create(ctx, a)).To(Succeed())
			Expect(writer.Create(ctx, b)).To(Succeed())
			Expect(writer.Delete(ctx, nil, a.Key, b.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(rack.MatchKeys(a.Key, b.Key)).
				Exists(ctx, tx)).To(BeFalse())
		})

		It("Should refuse to delete the embedded rack", func(ctx SpecContext) {
			Expect(writer.Delete(ctx, nil, svc.EmbeddedKey)).
				To(MatchError(ContainSubstring("cannot delete embedded rack")))
			Expect(svc.NewRetrieve().
				Where(rack.MatchKeys(svc.EmbeddedKey)).
				Exists(ctx, tx)).To(BeTrue())
		})

		It("Should refuse a rack that fails the guard", func(ctx SpecContext) {
			r := &rack.Rack{Name: "guarded"}
			Expect(writer.Create(ctx, r)).To(Succeed())
			refusal := errors.New("refused")
			Expect(writer.Delete(
				ctx,
				func(_ gorp.Context, _ rack.Rack) error { return refusal },
				r.Key,
			)).To(MatchError(refusal))
			Expect(svc.NewRetrieve().
				Where(rack.MatchKeys(r.Key)).
				Exists(ctx, tx)).To(BeTrue())
		})
	})

	Describe("Embedded Rack", func() {
		It("Should correctly create the node embedded rack", func(ctx SpecContext) {
			openTx()
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
				Expect(restartDB.WithTx(ctx, func(tx gorp.Tx) error {
					return before.rack.NewWriter(tx).Rename(ctx, key, "Renamed")
				})).To(Succeed())
				Expect(before.Close()).To(Succeed())
				after := DeferClose(openStack(ctx, restartDB, rack.ServiceConfig{}))
				Expect(after.rack.EmbeddedKey).To(Equal(key))
				var embedded []rack.Rack
				Expect(after.rack.NewRetrieve().
					Where(rack.MatchEmbedded(true)).
					Entries(&embedded).
					Exec(ctx, restartDB)).To(Succeed())
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
			"Should apply the actions to a new embedded rack before saving it",
			func(ctx SpecContext) {
				r := MustSucceed(svc.RetrieveOrCreateEmbedded(
					ctx,
					"edited embedded",
					rack.NewSetAction(rack.SetPayload{
						Path:  []string{"integrations"},
						Value: []string{"arc"},
					}),
				))
				var res rack.Rack
				Expect(svc.NewRetrieve().
					Where(rack.MatchKeys(r.Key)).
					Entry(&res).
					Exec(ctx, db)).To(Succeed())
				Expect(res.Name).To(Equal("edited embedded"))
				Expect(res.Embedded).To(BeTrue())
				Expect(res.Integrations).To(Equal([]string{"arc"}))
			},
		)
		It(
			"Should apply the actions to an existing embedded rack",
			func(ctx SpecContext) {
				rec := &Recorder[rack.Key, rack.Action]{}
				DeferCleanup(svc.Actions().OnChange(rec.Record))
				created := MustSucceed(
					svc.RetrieveOrCreateEmbedded(ctx, "kept embedded"),
				)
				set := rack.NewSetAction(rack.SetPayload{
					Path:  []string{"integrations"},
					Value: []string{"arc"},
				})
				r := MustSucceed(
					svc.RetrieveOrCreateEmbedded(ctx, "kept embedded", set),
				)
				Expect(r.Key).To(Equal(created.Key))
				Expect(r.Integrations).To(Equal([]string{"arc"}))
				frames := rec.Snapshot()
				Expect(frames).To(HaveLen(2))
				Expect(frames[0].Actions).To(Equal([]rack.Action{
					rack.NewCreateAction(rack.CreatePayload{Rack: created}),
				}))
				Expect(frames[1].Actions).To(Equal([]rack.Action{set}))
			},
		)
		It(
			"Should return a zero rack when the embedded rack fails to save",
			func(ctx SpecContext) {
				r, err := svc.RetrieveOrCreateEmbedded(
					ctx,
					"invalid embedded",
					rack.NewSetAction(
						rack.SetPayload{Path: []string{"name"}, Value: ""},
					),
				)
				Expect(err).To(MatchError(ContainSubstring("name: required")))
				Expect(r).To(Equal(rack.Rack{}))
			},
		)
	})

	Describe("Status", func() {
		It("Should initialize a rack with an unknown status", func(ctx SpecContext) {
			r := rack.Rack{Name: "test rack"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &r)
			})).To(Succeed())
			s := MustSucceed(retrieveStatusTx(ctx, stat, db, r.Key))
			Expect(s.Message).To(Equal("Rack status unknown"))
			Expect(s.Variant).To(Equal(status.VariantWarning))
			Expect(s.Time).To(BeNumerically("~", telem.Now(), 3*telem.SecondTS))
			Expect(s.Owner).To(HaveValue(Equal(r.ResourceID())))
		})

		It(
			"Should not write a provided status over the status of another rack",
			func(ctx SpecContext) {
				a := rack.Rack{Name: "rack a"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &a)
				})).To(Succeed())
				aKey := MustSucceed(retrieveStatusTx(ctx, stat, db, a.Key)).Key
				b := rack.Rack{Name: "rack b", Status: &rack.Status{
					Key:     aKey,
					Variant: status.VariantSuccess,
					Time:    telem.Now(),
					Message: "forged",
				}}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &b)
				})).To(Succeed())
				Expect(MustSucceed(retrieveStatusTx(ctx, stat, db, a.Key)).Message).
					To(Equal("Rack status unknown"))
				bStatus := MustSucceed(retrieveStatusTx(ctx, stat, db, b.Key))
				Expect(bStatus.Message).To(Equal("forged"))
				Expect(bStatus.Key).ToNot(Equal(aKey))
			},
		)

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
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &r)
				})).To(Succeed())
				s := MustSucceed(retrieveStatusTx(ctx, stat, db, r.Key))
				Expect(s.Message).To(Equal("Custom status message"))
				Expect(s.Description).To(Equal("Custom description"))
				Expect(s.Variant).To(Equal(status.VariantSuccess))
				Expect(s.Owner).To(HaveValue(Equal(r.ResourceID())))
				// Time should be auto-filled
				Expect(s.Time).To(BeNumerically("~", telem.Now(), 3*telem.SecondTS))
				// Name should be auto-filled from rack name
				Expect(s.Name).To(Equal(r.Name))
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
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &r)
				})).
					To(MatchError(ContainSubstring("variant")))
			},
		)

		It(
			"Should restore a missing status when the rack is renamed",
			func(ctx SpecContext) {
				openTx()
				r := rack.Rack{Name: "heal rack"}
				Expect(writer.Create(ctx, &r)).To(Succeed())

				sw := stat.NewWriter(tx)
				Expect(sw.DeleteOwned(ctx, r.ResourceID())).
					To(Succeed())
				Expect(retrieveStatusTx(ctx, stat, tx, r.Key)).Error().
					To(MatchError(query.ErrNotFound))

				Expect(writer.Rename(ctx, r.Key, "healed rack")).To(Succeed())

				healed := MustSucceed(retrieveStatusTx(ctx, stat, tx, r.Key))
				Expect(healed.Owner).To(HaveValue(Equal(r.ResourceID())))
			},
		)

		It(
			"Should keep the live status when a set action renames a rack",
			func(ctx SpecContext) {
				openTx()
				r := rack.Rack{
					Name: "starting rack",
					Status: &rack.Status{
						Variant: status.VariantWarning,
						Message: "Rack is starting",
						Time:    telem.SecondTS,
					},
				}
				Expect(writer.Create(ctx, &r)).To(Succeed())
				live := MustSucceed(retrieveStatusTx(ctx, stat, tx, r.Key))
				live.Variant = status.VariantSuccess
				live.Message = "Rack is connected"
				live.Details = rack.StatusDetails{Running: true}
				Expect(stat.NewWriter(tx).Set(ctx, &live)).To(Succeed())
				expected := live
				expected.Name = "connected rack"

				Expect(writer.Rename(ctx, r.Key, "connected rack")).To(Succeed())

				Expect(retrieveStatusTx(ctx, stat, tx, r.Key)).To(Equal(expected))
			},
		)

		It(
			"Should not write the status on an apply that keeps the rack name",
			func(ctx SpecContext) {
				frozenNow.Store(int64(telem.Now()))
				r := rack.Rack{Name: "unchanged rack"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &r)
				})).To(Succeed())
				var writes atomic.Int32
				DeferCleanup(stat.Observe().OnChange(func(
					_ context.Context,
					t gorp.TxReader[status.Key, status.Status[any]],
				) {
					for ch := range t {
						if o := ch.Value.Owner; o != nil && *o == r.ResourceID() {
							writes.Add(1)
						}
					}
				}))
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Apply(
						ctx,
						r.Key,
						rack.NewSetAction(rack.SetPayload{
							Path:  []string{"integrations"},
							Value: []string{"ni"},
						}),
					)
				})).To(Succeed())
				Consistently(
					writes.Load,
					50*telem.Millisecond.Duration(),
					5*telem.Millisecond.Duration(),
				).Should(BeZero())
			},
		)

		It(
			"Should mark a rack as dead when it doesn't receive a status within the health check interval",
			func(ctx SpecContext) {
				r := rack.Rack{Name: "dead test rack"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &r)
				})).To(Succeed())

				Eventually(func(g Gomega) {
					s := MustSucceed(retrieveStatusTx(ctx, stat, db, r.Key))
					g.Expect(s.Message).To(Equal("Driver not running"))
					g.Expect(s.Variant).To(Equal(status.VariantWarning))
					g.Expect(s.Time).
						To(BeNumerically("~", telem.Now(), 3*telem.SecondTS))
					g.Expect(s.Owner).To(HaveValue(Equal(r.ResourceID())))
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
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &r)
				})).To(Succeed())

				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return stat.NewWriter(tx).Set(ctx, &rack.Status{
						Owner:   new(r.ResourceID()),
						Name:    r.Name,
						Time:    telem.Now(),
						Variant: status.VariantSuccess,
						Message: "Running",
						Details: rack.StatusDetails{Running: true},
					})
				})).To(Succeed())

				Consistently(func(g Gomega) {
					s := MustSucceed(retrieveStatusTx(ctx, stat, db, r.Key))
					g.Expect(s.Message).To(Equal("Running"))
					g.Expect(s.Variant).To(Equal(status.VariantSuccess))
					g.Expect(s.Description).To(BeEmpty())
				}, 50*telem.Millisecond.Duration(), 5*telem.Millisecond.Duration()).Should(Succeed())
			},
		)

		DescribeTable(
			"Should count a rack status as a heartbeat only when the Driver is running",
			func(ctx SpecContext, variant status.Variant, running bool) {
				name := CurrentSpecReport().LeafNodeText
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
				DeferCleanup(
					svc.OnSuspect(func(_ context.Context, s rack.Status) error {
						if s.Name == name {
							alerts.Add(1)
						}
						return nil
					}),
				)
				now := telem.Now()
				frozenNow.Store(int64(now))
				r := rack.Rack{Name: name}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &r)
				})).To(Succeed())
				// Each beat advances the clock half an interval, so only beats that
				// count as alive keep the rack under one interval of silence.
				for range 4 {
					now = now.Add(svc.HealthCheckInterval / 2)
					frozenNow.Store(int64(now))
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return stat.NewWriter(tx).Set(ctx, &rack.Status{
							Owner:   new(r.ResourceID()),
							Name:    r.Name,
							Time:    now,
							Variant: variant,
							Message: "Heartbeat",
							Details: rack.StatusDetails{Running: running},
						})
					})).To(Succeed())
				}

				if !running {
					Eventually(writes.Load).Should(BeEquivalentTo(1))
					Eventually(alerts.Load).Should(BeEquivalentTo(1))
					return
				}
				Consistently(func(g Gomega) {
					g.Expect(writes.Load()).To(BeZero())
					g.Expect(alerts.Load()).To(BeZero())
					s := MustSucceed(retrieveStatusTx(ctx, stat, db, r.Key))
					g.Expect(s.Variant).To(Equal(variant))
				}, 100*time.Millisecond, 5*time.Millisecond).Should(Succeed())
			},
			Entry("an error from a running Driver", status.VariantError, true),
			Entry("a warning from a running Driver", status.VariantWarning, true),
			Entry("a warning from the Core", status.VariantWarning, false),
			Entry("a success without a running Driver", status.VariantSuccess, false),
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
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &r)
			})).To(Succeed())

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
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &r)
				})).To(Succeed())

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
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &r)
				})).To(Succeed())

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
						if *s.Owner == r.ResourceID() {
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
						Owner:   new(r.ResourceID()),
						Name:    r.Name,
						Time:    telem.Now(),
						Variant: status.VariantSuccess,
						Message: "Running",
						Details: rack.StatusDetails{Running: true},
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
			"Should not extend the deadline of a silent rack on a rename",
			func(ctx SpecContext) {
				start := telem.Now()
				frozenNow.Store(int64(start))
				r := rack.Rack{
					Name: "silent rack",
					Status: &rack.Status{
						Variant: status.VariantSuccess,
						Message: "Driver is running",
						Time:    start,
					},
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &r)
				})).To(Succeed())
				frozenNow.Store(int64(start.Add(6 * telem.Millisecond)))
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Rename(ctx, r.Key, "renamed silent rack")
				})).To(Succeed())
				frozenNow.Store(int64(start.Add(12 * telem.Millisecond)))
				Eventually(func(g Gomega) {
					s := MustSucceed(retrieveStatusTx(ctx, stat, db, r.Key))
					g.Expect(s.Message).To(Equal("Driver not running"))
					g.Expect(s.Name).To(Equal("renamed silent rack"))
				}).Should(Succeed())
			},
		)

		It(
			"Should report a silent rack once a rename in flight commits",
			func(ctx SpecContext) {
				start := telem.Now()
				frozenNow.Store(int64(start))
				r := rack.Rack{
					Name: "overwritten rack",
					Status: &rack.Status{
						Variant: status.VariantSuccess,
						Message: "Driver is running",
						Time:    telem.SecondTS,
					},
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &r)
				})).To(Succeed())
				renameTx := DeferClose(db.OpenTx())
				Expect(svc.NewWriter(renameTx).Rename(ctx, r.Key, "renamed")).
					To(Succeed())
				frozenNow.Store(int64(start.Add(12 * telem.Millisecond)))
				Expect(renameTx.Commit(ctx)).To(Succeed())
				Eventually(func(g Gomega) {
					s := MustSucceed(retrieveStatusTx(ctx, stat, db, r.Key))
					g.Expect(s.Message).To(Equal("Driver not running"))
					g.Expect(s.Name).To(Equal("renamed"))
				}).Should(Succeed())
			},
		)

		It(
			"Should not extend the deadline of a silent rack after a failed outage write",
			func(ctx SpecContext) {
				start := telem.Now()
				var (
					now          atomic.Int64
					failed       atomic.Bool
					ticksAfter   atomic.Int32
					outageStatus = []byte("Driver not running")
				)
				now.Store(int64(start))
				failDB := DeferClose(gorp.Wrap(failingDB{
					DB: memkv.New(),
					fail: func(values [][]byte) error {
						for _, v := range values {
							if bytes.Contains(v, outageStatus) &&
								failed.CompareAndSwap(false, true) {
								now.Store(int64(start.Add(5 * telem.Millisecond)))
								return errors.New("commit failed")
							}
						}
						return nil
					},
				}))
				failing := DeferClose(openStack(ctx, failDB, rack.ServiceConfig{
					HealthCheckInterval: 10 * telem.Millisecond,
					Now: func() telem.TimeStamp {
						if failed.Load() {
							ticksAfter.Add(1)
						}
						return telem.TimeStamp(now.Load())
					},
				}))
				r := rack.Rack{
					Name: "failed outage rack",
					Status: &rack.Status{
						Variant: status.VariantSuccess,
						Message: "Driver is running",
						Time:    telem.SecondTS,
					},
				}
				Expect(failDB.WithTx(ctx, func(tx gorp.Tx) error {
					return failing.rack.NewWriter(tx).Create(ctx, &r)
				})).To(Succeed())
				now.Store(int64(start.Add(12 * telem.Millisecond)))
				// The next tick shows that the monitor finished with the failed write.
				Eventually(ticksAfter.Load).Should(BeNumerically(">", 0))

				Expect(failDB.WithTx(ctx, func(tx gorp.Tx) error {
					return failing.rack.NewWriter(tx).Rename(ctx, r.Key, "renamed")
				})).
					To(Succeed())
				now.Store(int64(start.Add(12 * telem.Millisecond)))
				Eventually(func(g Gomega) {
					s := MustSucceed(
						retrieveStatusTx(ctx, failing.status, failDB, r.Key),
					)
					g.Expect(s.Message).To(Equal("Driver not running"))
					g.Expect(s.Name).To(Equal("renamed"))
				}).Should(Succeed())
			},
		)

		DescribeTable(
			"Should rename a rack that holds any stored status",
			func(ctx SpecContext, codec encoding.Codec, details any) {
				kv := memkv.New()
				restartDB := DeferClose(gorp.Wrap(kv))
				before := openStack(ctx, restartDB, rack.ServiceConfig{})
				owner := before.rack.EmbeddedKey.ResourceID()
				var held status.Status[any]
				Expect(before.status.NewRetrieve[any]().
					Where(status.MatchOwners[any](owner)).
					Entry(&held).
					Exec(ctx, restartDB)).To(Succeed())
				Expect(before.Close()).To(Succeed())
				// Migrations do not run again, so the status stays in this encoding.
				Expect(gorp.NewCreate[status.Key, status.Status[any]]().
					Entry(&status.Status[any]{
						Key:     held.Key,
						Owner:   &owner,
						Name:    "stale name",
						Variant: status.VariantSuccess,
						Time:    telem.Now(),
						Details: details,
					}).
					Exec(ctx, gorp.Wrap(kv, gorp.WithCodec(codec)))).To(Succeed())

				after := DeferClose(openStack(ctx, restartDB, rack.ServiceConfig{}))
				Expect(restartDB.WithTx(ctx, func(tx gorp.Tx) error {
					return after.rack.NewWriter(tx).
						Rename(ctx, after.rack.EmbeddedKey, "renamed")
				})).To(Succeed())
				var res status.Status[any]
				Expect(after.status.NewRetrieve[any]().
					Where(status.MatchOwners[any](owner)).
					Entry(&res).
					Exec(ctx, restartDB)).To(Succeed())
				Expect(res.Name).To(Equal("renamed"))
				Expect(res.Details).To(Equal(details))
			},
			Entry("ORC with string details", orc.Codec, "running"),
			Entry("ORC with array details", orc.Codec, []any{"running"}),
			Entry("ORC with object details", orc.Codec, map[string]any{"rack": 1.5}),
			Entry("msgpack with string details", msgpack.Codec, "running"),
			Entry("msgpack with array details", msgpack.Codec, []any{"running"}),
			Entry(
				"msgpack with object details",
				msgpack.Codec,
				map[string]any{"rack": 1.5},
			),
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
				Expect(restartDB.WithTx(ctx, func(tx gorp.Tx) error {
					return before.rack.NewWriter(tx).Create(ctx, &r)
				})).To(Succeed())
				Expect(before.Close()).To(Succeed())

				after := DeferClose(openStack(ctx, restartDB, rack.ServiceConfig{
					HealthCheckInterval: 10 * telem.Millisecond,
				}))
				Eventually(func(g Gomega) {
					s := MustSucceed(
						retrieveStatusTx(ctx, after.status, restartDB, r.Key),
					)
					g.Expect(s.Variant).To(Equal(status.VariantWarning))
					g.Expect(s.Message).To(Equal("Driver not running"))
				}).Should(Succeed())
			},
		)
	})
})

var _ = Describe("Group", func() {
	var (
		db *gorp.DB
		s  stack
		tx gorp.Tx
		w  rack.Writer
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		s = DeferClose(openStack(ctx, db, rack.ServiceConfig{}))
		tx = DeferClose(db.OpenTx())
		w = s.rack.NewWriter(tx)
	})
	createGroup := func(ctx context.Context, g group.Group) group.Group {
		GinkgoHelper()
		Expect(s.group.NewWriter(tx).Create(ctx, &g)).To(Succeed())
		return g
	}
	retrieveGroup := func(ctx context.Context, key rack.Key) *group.Key {
		GinkgoHelper()
		var r rack.Rack
		Expect(s.rack.NewRetrieve().
			Where(rack.MatchKeys(key)).
			Entry(&r).
			Exec(ctx, tx)).To(Succeed())
		return r.Group
	}

	It(
		"Should move a rack into a top-level device group and back out",
		func(ctx SpecContext) {
			g := createGroup(ctx, group.Group{Name: "Top", Scope: resource.TypeDevice})
			r := rack.Rack{Name: "grouped"}
			Expect(w.Create(ctx, &r)).To(Succeed())
			Expect(w.SetGroup(ctx, &g.Key, r.Key)).To(Succeed())
			Expect(retrieveGroup(ctx, r.Key)).To(Equal(&g.Key))
			Expect(w.SetGroup(ctx, nil, r.Key)).To(Succeed())
			Expect(retrieveGroup(ctx, r.Key)).To(BeNil())
		},
	)

	It("Should reject a group under a rack", func(ctx SpecContext) {
		g := createGroup(ctx, group.Group{
			Name:   "Under rack",
			Parent: new(s.rack.EmbeddedKey.ResourceID()),
		})
		r := rack.Rack{Name: "nested"}
		Expect(w.Create(ctx, &r)).To(Succeed())
		Expect(w.SetGroup(ctx, &g.Key, r.Key)).To(MatchError(
			"group: a rack cannot sit in a group under rack " +
				s.rack.EmbeddedKey.ResourceID().Key + ": validation error",
		))
		Expect(retrieveGroup(ctx, r.Key)).To(BeNil())
	})

	It("Should keep the stored group when a rack is renamed", func(ctx SpecContext) {
		g := createGroup(ctx, group.Group{Name: "Kept", Scope: resource.TypeDevice})
		r := rack.Rack{Name: "kept", Group: &g.Key}
		Expect(w.Create(ctx, &r)).To(Succeed())
		Expect(w.Rename(ctx, r.Key, "kept again")).To(Succeed())
		Expect(retrieveGroup(ctx, r.Key)).To(Equal(&g.Key))
	})

	It(
		"Should reject a created rack in a group of another scope",
		func(ctx SpecContext) {
			g := createGroup(ctx, group.Group{
				Name:  "Channels",
				Scope: resource.TypeChannel,
			})
			r := rack.Rack{Name: "mismatch", Group: &g.Key}
			Expect(w.Create(ctx, &r)).To(MatchError(
				"group: a rack cannot sit in a channel group: validation error",
			))
		},
	)
})

var _ = Describe("Writer in concurrent transactions", func() {
	var (
		db *gorp.DB
		s  stack
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		s = DeferClose(openStack(ctx, db, rack.ServiceConfig{}))
	})
	createGroup := func(ctx context.Context, g group.Group) group.Group {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return s.group.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		return g
	}
	retrieveGroup := func(ctx context.Context, key rack.Key) *group.Key {
		GinkgoHelper()
		var r rack.Rack
		Expect(s.rack.NewRetrieve().
			Where(rack.MatchKeys(key)).
			Entry(&r).
			Exec(ctx, db)).To(Succeed())
		return r.Group
	}

	It(
		"Should keep the group that a move writes during concurrent applies",
		func(ctx SpecContext) {
			groups := make([]group.Group, 50)
			for i := range groups {
				groups[i] = createGroup(ctx, group.Group{
					Name:  fmt.Sprintf("Concurrent %d", i),
					Scope: resource.TypeDevice,
				})
			}
			r := rack.Rack{Name: "concurrent"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return s.rack.NewWriter(tx).Create(ctx, &r)
			})).To(Succeed())
			var (
				wg    sync.WaitGroup
				moved atomic.Bool
			)
			for range 4 {
				wg.Go(func() {
					defer GinkgoRecover()
					for !moved.Load() {
						Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
							return s.rack.NewWriter(tx).Rename(ctx, r.Key, "renamed")
						})).To(Succeed())
					}
				})
			}
			wg.Go(func() {
				defer GinkgoRecover()
				defer moved.Store(true)
				for _, g := range groups {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return s.rack.NewWriter(tx).SetGroup(ctx, &g.Key, r.Key)
					})).
						To(Succeed())
					Expect(retrieveGroup(ctx, r.Key)).To(Equal(&g.Key))
				}
			})
			wg.Wait()
			Expect(retrieveGroup(ctx, r.Key)).To(Equal(&groups[len(groups)-1].Key))
		},
	)

	It(
		"Should not put a rack in a group that moves under another rack at once",
		func(ctx SpecContext) {
			for range 30 {
				g := createGroup(ctx, group.Group{
					Name:  "Moving",
					Scope: resource.TypeDevice,
				})
				r := rack.Rack{Name: "joining"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return s.rack.NewWriter(tx).Create(ctx, &r)
				})).To(Succeed())
				var (
					wg               sync.WaitGroup
					joinErr, moveErr error
				)
				wg.Go(func() {
					joinErr = db.WithTx(ctx, func(tx gorp.Tx) error {
						return s.rack.NewWriter(tx).SetGroup(ctx, &g.Key, r.Key)
					})
				})
				wg.Go(func() {
					moveErr = db.WithTx(ctx, func(tx gorp.Tx) error {
						return s.group.NewWriter(tx).
							SetParent(ctx, new(s.rack.EmbeddedKey.ResourceID()), g.Key)
					})
				})
				wg.Wait()
				Expect([]error{joinErr, moveErr}).To(ContainElement(HaveOccurred()))
			}
		},
	)
})
