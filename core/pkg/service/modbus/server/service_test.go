// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/modbus/server"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Service", func() {
	var (
		stat    *status.Service
		rackSvc *rack.Service
		svc     *server.Service
		tx      gorp.Tx
	)
	BeforeEach(func(ctx SpecContext) {
		MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		stat = MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: db}))
		rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:         db,
			HostKey:    1,
			Status:     stat,
			DriverRole: MustSucceed(racktestutil.CreateDriverRole(ctx, db, roleSvc)),
		}))
		svc = MustOpen(server.OpenService(ctx, server.ServiceConfig{
			DB:     db,
			Status: stat,
		}))
		tx = DeferClose(db.OpenTx())
	})

	newServer := func() server.Server {
		return server.Server{
			Rack:          rackSvc.EmbeddedKey,
			Name:          "Boiler PLC",
			Connection:    tcp("boiler"),
			RequestPolicy: requestPolicy(),
		}
	}
	create := func(ctx context.Context) server.Server {
		GinkgoHelper()
		s := newServer()
		Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
		return s
	}
	retrieve := func(ctx context.Context, key server.Key) server.Server {
		GinkgoHelper()
		var s server.Server
		Expect(svc.NewRetrieve().
			Where(server.MatchKeys(key)).
			Entry(&s).
			Exec(ctx, tx)).To(Succeed())
		return s
	}
	retrieveStatus := func(ctx context.Context, key server.Key) server.Status {
		GinkgoHelper()
		var s server.Status
		Expect(stat.NewRetrieve[server.StatusDetails]().
			Where(status.MatchOwners[server.StatusDetails](server.ResourceID(key))).
			Entry(&s).
			Exec(ctx, tx)).To(Succeed())
		return s
	}
	identity := func(s server.Server) server.Identity {
		return server.Identity{
			Rack:          s.Rack,
			Name:          s.Name,
			Connection:    s.Connection,
			RequestPolicy: s.RequestPolicy,
		}
	}

	Describe("OpenService", func() {
		DescribeTable("Should reject a config missing a required field",
			func(ctx SpecContext, field string, unset func(*server.ServiceConfig)) {
				cfg := server.ServiceConfig{DB: db, Status: stat}
				unset(&cfg)
				Expect(server.OpenService(ctx, cfg)).Error().
					To(MatchError(ContainSubstring(field + ": must be non-nil")))
			},
			Entry("db", "db", func(c *server.ServiceConfig) { c.DB = nil }),
			Entry("status", "status", func(c *server.ServiceConfig) { c.Status = nil }),
		)

		It("Should let a later config override an earlier one", func() {
			cfg := server.ServiceConfig{DB: db}.Override(server.ServiceConfig{
				Status: stat,
			})
			Expect(cfg.DB).To(BeIdenticalTo(db))
			Expect(cfg.Status).To(BeIdenticalTo(stat))
			Expect(cfg.Validate()).To(Succeed())
		})
	})

	Describe("Create", func() {
		It("Should mint a key, store the server, and write an unknown status", func(
			ctx SpecContext,
		) {
			s := create(ctx)
			Expect(s.Key).ToNot(Equal(uuid.Nil()))
			Expect(retrieve(ctx, s.Key)).To(Equal(s))
			st := retrieveStatus(ctx, s.Key)
			Expect(st.Variant).To(Equal(status.VariantWarning))
			Expect(st.Message).To(Equal("Status unknown"))
			Expect(st.Name).To(Equal(s.Name))
			Expect(st.Details.Server).To(Equal(s.Key))
			Expect(st.Details.Rack).To(Equal(s.Rack))
		})

		It("Should accept one register read into two indexes", func(ctx SpecContext) {
			s := newServer()
			s.Reads = []server.Read{read(5, 1, 2), read(5, 3, 4)}
			Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
		})

		It("Should keep a caller-set key", func(ctx SpecContext) {
			s := newServer()
			s.Key = uuid.New()
			Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
			Expect(retrieve(ctx, s.Key).Key).To(Equal(s.Key))
		})

		It("Should give an existing status the name and rack of a replaced server",
			func(ctx SpecContext) {
				s := create(ctx)
				other := rack.Rack{Name: "Other"}
				Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
				s.Name = "Pump PLC"
				s.Rack = other.Key
				Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
				st := retrieveStatus(ctx, s.Key)
				Expect(st.Name).To(Equal("Pump PLC"))
				Expect(st.Details.Rack).To(Equal(other.Key))
			},
		)

		DescribeTable("Should reject an invalid server",
			func(ctx SpecContext, change func(*server.Server)) {
				s := newServer()
				change(&s)
				Expect(svc.NewWriter(tx).Create(ctx, &s)).
					To(MatchError(validate.ErrValidation))
			},
			Entry("no name", func(s *server.Server) { s.Name = "" }),
			Entry("no transport", func(s *server.Server) {
				s.Connection = server.Connection{}
			}),
			Entry("a baud rate the Driver cannot set", func(s *server.Server) {
				s.Connection = rtu(14400)
			}),
			Entry("a missing rack", func(s *server.Server) { s.Rack = 999 }),
			Entry("two reads of one register into one index", func(s *server.Server) {
				s.Reads = []server.Read{read(5, 1, 2), read(5, 1, 3)}
			}),
			Entry("two writes of one coil", func(s *server.Server) {
				s.Writes = []server.Write{write(1, 2), write(1, 3)}
			}),
			Entry("a host with a path", func(s *server.Server) {
				s.Connection = tcp("plc/1")
			}),
			Entry("a host that ends in a number", func(s *server.Server) {
				s.Connection = tcp("10.0.0.256")
			}),
			Entry("a gap wider than one read", func(s *server.Server) {
				s.RequestPolicy.MaxReadRegisters = 10
				s.RequestPolicy.MaxGap = 11
			}),
		)

		DescribeTable("Should reject writes that set one register twice",
			func(ctx SpecContext, a, b server.WriteValueVariant, offset uint16) {
				s := newServer()
				s.Writes = []server.Write{
					holdingWrite(5, a, 2),
					holdingWrite(5+offset, b, 3),
				}
				Expect(svc.NewWriter(tx).Create(ctx, &s)).
					To(MatchError(ContainSubstring("another write sets")))
			},
			Entry("two data types at one address",
				whole(server.DataTypeFloat32),
				whole(server.DataTypeUint16), uint16(0)),
			Entry("a float32 and the next register",
				whole(server.DataTypeFloat32),
				whole(server.DataTypeUint16), uint16(1)),
			Entry("a whole value and one of its bits",
				whole(server.DataTypeUint16),
				bit(3), uint16(0)),
			Entry("one bit twice",
				bit(3), bit(3), uint16(0)),
		)

		It("Should accept writes of two bits of one register", func(ctx SpecContext) {
			s := newServer()
			s.Writes = []server.Write{
				holdingWrite(5, bit(1), 2),
				holdingWrite(5, bit(2), 3),
				holdingWrite(6, whole(server.DataTypeInt16), 4),
			}
			Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
		})

		DescribeTable("Should accept a host",
			func(ctx SpecContext, host string) {
				s := newServer()
				s.Connection = tcp(host)
				Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
			},
			Entry("hostname", "plc-1.plant.local"),
			Entry("IPv4", "10.0.0.5"),
			Entry("IPv6", "fe80::1"),
		)

		It("Should accept two reads of one address as different values",
			func(ctx SpecContext) {
				s := newServer()
				bit := read(5, 1, 3)
				bit.Location = server.Location{Variant: server.HoldingRegisterLocation{
					Address: 5,
					Value:   server.Value{Variant: server.BitValue{Bit: 2}},
				}}
				s.Reads = []server.Read{read(5, 1, 2), bit}
				Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
			},
		)
	})

	Describe("Apply", func() {
		It("Should write the identity group and keep the channel maps", func(
			ctx SpecContext,
		) {
			s := newServer()
			s.Reads = []server.Read{read(5, 4, 5)}
			Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
			update := identity(s)
			update.Name = "Renamed"
			update.Connection = rtu(9600)
			update.RequestPolicy.Delay = 10 * telem.Millisecond
			Expect(svc.NewWriter(tx).Apply(ctx, s.Key, server.Update{
				Variant: server.IdentityUpdate{Identity: update},
			})).To(Succeed())
			got := retrieve(ctx, s.Key)
			Expect(got.Name).To(Equal("Renamed"))
			Expect(got.Connection).To(Equal(rtu(9600)))
			Expect(got.RequestPolicy.Delay).To(Equal(10 * telem.Millisecond))
			Expect(got.Reads).To(Equal(s.Reads))
			Expect(retrieveStatus(ctx, s.Key).Name).To(Equal("Renamed"))
		})

		It("Should write the configuration group and keep the connection", func(
			ctx SpecContext,
		) {
			s := create(ctx)
			Expect(svc.NewWriter(tx).Apply(ctx, s.Key, server.Update{
				Variant: server.ConfigurationUpdate{Configuration: server.Configuration{
					Reads:  []server.Read{read(1, 1, 2)},
					Writes: []server.Write{write(2, 3)},
				}},
			})).To(Succeed())
			got := retrieve(ctx, s.Key)
			Expect(got.Name).To(Equal(s.Name))
			Expect(got.Connection).To(Equal(s.Connection))
			Expect(got.RequestPolicy).To(Equal(s.RequestPolicy))
			Expect(got.Reads).To(Equal([]server.Read{read(1, 1, 2)}))
			Expect(got.Writes).To(Equal([]server.Write{write(2, 3)}))
		})

		It("Should move the status with the rack", func(ctx SpecContext) {
			s := create(ctx)
			other := rack.Rack{Name: "Other"}
			Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			update := identity(s)
			update.Rack = other.Key
			Expect(svc.NewWriter(tx).Apply(ctx, s.Key, server.Update{
				Variant: server.IdentityUpdate{Identity: update},
			})).To(Succeed())
			Expect(retrieveStatus(ctx, s.Key).Details.Rack).To(Equal(other.Key))
		})

		It("Should write an unknown status when the status is missing", func(
			ctx SpecContext,
		) {
			s := create(ctx)
			Expect(stat.NewWriter(tx).Delete(ctx, retrieveStatus(ctx, s.Key).Key)).
				To(Succeed())
			update := identity(s)
			update.Name = "Renamed"
			Expect(svc.NewWriter(tx).Apply(ctx, s.Key, server.Update{
				Variant: server.IdentityUpdate{Identity: update},
			})).To(Succeed())
			st := retrieveStatus(ctx, s.Key)
			Expect(st.Message).To(Equal("Status unknown"))
			Expect(st.Name).To(Equal("Renamed"))
		})

		DescribeTable(
			"Should reject an invalid update",
			func(ctx SpecContext, update func(server.Server) server.Update) {
				s := create(ctx)
				Expect(svc.NewWriter(tx).Apply(ctx, s.Key, update(s))).
					To(MatchError(validate.ErrValidation))
				Expect(retrieve(ctx, s.Key)).To(Equal(s))
			},
			Entry("no group", func(server.Server) server.Update {
				return server.Update{}
			}),
			Entry("an identity on rack 0", func(s server.Server) server.Update {
				i := identity(s)
				i.Rack = 0
				return server.Update{Variant: server.IdentityUpdate{Identity: i}}
			}),
			Entry("an identity without a name", func(s server.Server) server.Update {
				i := identity(s)
				i.Name = ""
				return server.Update{Variant: server.IdentityUpdate{Identity: i}}
			}),
			Entry(
				"a baud rate the Driver cannot set",
				func(s server.Server) server.Update {
					i := identity(s)
					i.Connection = rtu(14400)
					return server.Update{Variant: server.IdentityUpdate{Identity: i}}
				},
			),
		)

		It("Should return not found for a missing server", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Apply(ctx, uuid.New(), server.Update{
				Variant: server.ConfigurationUpdate{},
			})).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Delete", func() {
		It("Should remove the server and its status", func(ctx SpecContext) {
			s := create(ctx)
			Expect(svc.NewWriter(tx).Delete(ctx, s.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(server.MatchKeys(s.Key)).
				Exists(ctx, tx)).To(BeFalse())
			var st server.Status
			Expect(stat.NewRetrieve[server.StatusDetails]().
				Where(status.MatchOwners[server.StatusDetails](server.ResourceID(s.Key))).
				Entry(&st).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})

		It("Should be idempotent", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Delete(ctx, uuid.New())).To(Succeed())
		})
	})

	It("Should refuse to delete a rack that holds a server", func(ctx SpecContext) {
		other := rack.Rack{Name: "Other"}
		Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
		s := newServer()
		s.Rack = other.Key
		Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
		Expect(rackSvc.NewWriter(tx).Delete(ctx, nil, other.Key)).To(MatchError(
			ContainSubstring("because modbus_server %s references it", s.Key),
		))
		Expect(retrieve(ctx, s.Key).Rack).To(Equal(other.Key))
	})

	Describe("Resources", func() {
		It("Should name a server by its resource ID", func(ctx SpecContext) {
			s := create(ctx)
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{s.ResourceID()})).
				To(Equal([]resource.Resource{{ID: s.ResourceID(), Name: s.Name}}))
		})

		It("Should find a server by its name", func(ctx SpecContext) {
			s := create(ctx)
			res := MustSucceed(svc.Resources().Search(
				ctx,
				tx,
				search.Query[resource.ID]{Term: "boiler", Limit: 10},
			))
			Expect(res.Keys).To(ContainElement(s.ResourceID()))
		})

		It("Should reject a resource ID with a malformed key", func(ctx SpecContext) {
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{{
				Type: resource.TypeModbusServer,
				Key:  "plc",
			}})).Error().To(MatchError(validate.ErrValidation))
		})
	})
})
