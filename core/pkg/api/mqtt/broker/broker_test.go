// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package broker_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	apibroker "github.com/synnaxlabs/synnax/pkg/api/mqtt/broker"
	"github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/broker"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var (
	brokerType = resource.ID{Type: resource.TypeMqttBroker}
	statusType = resource.ID{Type: resource.TypeStatus}
	allWrites  = []access.Action{
		access.ActionCreate,
		access.ActionUpdate,
		access.ActionDelete,
	}
)

var _ = Describe("Service", func() {
	var u user.User
	BeforeEach(func(ctx SpecContext) { u = createUser(ctx) })

	newBroker := func(overrides ...func(*broker.Broker)) broker.Broker {
		var s broker.Broker
		s.SetDefaults()
		s.Key = uuid.New()
		s.Rack = svc.Rack.EmbeddedKey
		s.Name = "API-" + uuid.New().String()
		s.Host = "mqtt.example.com"
		for _, override := range overrides {
			override(&s)
		}
		return s
	}
	grantRetrieve := func(ctx SpecContext) {
		GinkgoHelper()
		grant(ctx, u, []access.Action{access.ActionRetrieve}, brokerType)
	}
	// grantWrites lets u write any MQTT broker, its status, and the rack it sits on.
	grantWrites := func(ctx SpecContext, r rack.Key) {
		GinkgoHelper()
		grant(ctx, u, allWrites, brokerType, statusType)
		grant(ctx, u, []access.Action{access.ActionUpdate}, r.ResourceID())
	}
	retrieveBroker := func(ctx SpecContext, key broker.Key) broker.Broker {
		GinkgoHelper()
		var s broker.Broker
		Expect(svc.MQTT.Brokers.NewRetrieve().
			Where(broker.MatchKeys(key)).
			Entry(&s).
			Exec(ctx, db)).To(Succeed())
		return s
	}
	createBroker := func(ctx SpecContext, overrides ...func(*broker.Broker)) broker.Broker {
		GinkgoHelper()
		s := newBroker(overrides...)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.MQTT.Brokers.NewWriter(tx).Create(ctx, &s)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.MQTT.Brokers.NewWriter(tx).Delete(ctx, s.Key)
			})).To(Succeed())
		})
		return s
	}

	Describe("NewService", func() {
		DescribeTable("Should reject a config missing a required field",
			func(field string, unset func(*apibroker.ServiceConfig)) {
				cfg := apibroker.ServiceConfig{
					Broker: svc.MQTT.Brokers,
					Status: svc.Status,
				}
				unset(&cfg)
				Expect(apibroker.NewService(cfg)).Error().
					To(MatchError(ContainSubstring(field + ": must be non-nil")))
			},
			Entry("broker", "broker", func(c *apibroker.ServiceConfig) {
				c.Broker = nil
			}),
			Entry("status", "status", func(c *apibroker.ServiceConfig) {
				c.Status = nil
			}),
		)

		It("Should let a later config override an earlier one", func() {
			base := apibroker.ServiceConfig{Broker: svc.MQTT.Brokers}
			Expect(apibroker.NewService(base, apibroker.ServiceConfig{
				Status: svc.Status,
			})).ToNot(BeNil())
		})
	})

	Describe("Retrieve", func() {
		It("Should reject a named broker when retrieve is not granted", func(
			ctx SpecContext,
		) {
			key := createBroker(ctx).Key
			Expect(retrieve(
				testutil.AuthedCtx(ctx, u),
				apibroker.RetrieveRequest{
					Where: &broker.Where{Key: &filter.Set[broker.Key]{Eq: &key}},
				},
			)).Error().To(MatchError(access.ErrDenied))
		})

		It("Should filter by rack and attach the status", func(ctx SpecContext) {
			s := createBroker(ctx)
			createBroker(ctx)
			st := broker.Status{
				Owner:   new(s.ResourceID()),
				Name:    s.Name,
				Time:    telem.Now(),
				Variant: status.VariantSuccess,
				Message: "Broker connected",
				Details: broker.StatusDetails{Broker: s.Key, Connected: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Status.NewWriter(tx).Set(ctx, &st)
			})).To(Succeed())
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apibroker.RetrieveRequest{
					Where: &broker.Where{
						Name: &filter.String{Eq: &s.Name},
						Rack: &filter.Set[rack.Key]{Eq: &s.Rack},
					},
					IncludeStatus: true,
				},
			))
			Expect(res.Brokers).To(HaveLen(1))
			Expect(res.Brokers[0].Key).To(Equal(s.Key))
			Expect(res.Brokers[0].Status.Message).To(Equal("Broker connected"))
			Expect(res.Brokers[0].Status.Details.Connected).To(BeTrue())
		})

		It("Should leave out the brokers the subject cannot retrieve", func(
			ctx SpecContext,
		) {
			name := createBroker(ctx).Name
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apibroker.RetrieveRequest{
					Where: &broker.Where{Name: &filter.String{Eq: &name}},
				},
			))
			Expect(res.Brokers).To(BeEmpty())
		})

		It("Should find a broker by its name", func(ctx SpecContext) {
			d := createBroker(ctx, func(s *broker.Broker) { s.Name = "Boiler PLC" })
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apibroker.RetrieveRequest{SearchTerm: "boiler plc"},
			))
			Expect(lo.Map(res.Brokers, func(s broker.Broker, _ int) broker.Key {
				return s.Key
			})).To(ContainElement(d.Key))
		})

		It("Should cap the results at the limit", func(ctx SpecContext) {
			a := createBroker(ctx)
			b := createBroker(ctx)
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apibroker.RetrieveRequest{
					Where: &broker.Where{Key: &filter.Set[broker.Key]{
						In: []broker.Key{a.Key, b.Key},
					}},
					Limit: new(uint32(1)),
				},
			))
			Expect(res.Brokers).To(HaveLen(1))
		})

		It("Should page past the after key", func(ctx SpecContext) {
			a := createBroker(ctx)
			b := createBroker(ctx)
			grantRetrieve(ctx)
			authed := testutil.AuthedCtx(ctx, u)
			req := apibroker.RetrieveRequest{
				Where: &broker.Where{Key: &filter.Set[broker.Key]{
					In: []broker.Key{a.Key, b.Key},
				}},
				Limit: new(uint32(1)),
			}
			first := MustSucceed(retrieve(authed, req))
			Expect(first.Brokers).To(HaveLen(1))
			req.After = &apibroker.Cursor{Key: first.Brokers[0].Key}
			second := MustSucceed(retrieve(authed, req))
			Expect(second.Brokers).To(HaveLen(1))
			Expect(second.Brokers[0].Key).
				ToNot(Equal(first.Brokers[0].Key))
		})

		It("Should reject a search that resumes after a key", func(ctx SpecContext) {
			d := createBroker(ctx)
			grantRetrieve(ctx)
			Expect(retrieve(
				testutil.AuthedCtx(ctx, u),
				apibroker.RetrieveRequest{
					SearchTerm: d.Name,
					After:      &apibroker.Cursor{Key: d.Key},
				},
			)).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should return a path error for a malformed where", func(ctx SpecContext) {
			grantRetrieve(ctx)
			Expect(retrieve(testutil.AuthedCtx(ctx, u), apibroker.RetrieveRequest{
				Where: &broker.Where{Name: &filter.String{Regex: new("(a")}},
			})).Error().To(SatisfyAll(
				BeAssignableToTypeOf(validate.PathError{}),
				MatchError(ContainSubstring("where.name.regex: error parsing regexp")),
				MatchError(validate.ErrValidation),
			))
		})

		It("Should return no brokers for a zero limit", func(ctx SpecContext) {
			createBroker(ctx)
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apibroker.RetrieveRequest{Limit: new(uint32(0))},
			))
			Expect(res.Brokers).To(BeEmpty())
		})

		It("Should return no brokers for a missing key", func(ctx SpecContext) {
			grantRetrieve(ctx)
			missing := uuid.New()
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apibroker.RetrieveRequest{
					Where: &broker.Where{Key: &filter.Set[broker.Key]{Eq: &missing}},
				},
			))
			Expect(res.Brokers).To(BeEmpty())
		})
	})

	Describe("Create", func() {
		It("Should mint a key when writes are granted", func(ctx SpecContext) {
			grantWrites(ctx, svc.Rack.EmbeddedKey)
			s := newBroker()
			s.Key = uuid.Nil()
			res := MustSucceed(create(
				testutil.AuthedCtx(ctx, u),
				apibroker.CreateRequest{Brokers: []broker.Broker{s}},
			))
			key := res.Brokers[0].Key
			Expect(key).ToNot(Equal(uuid.Nil()))
			Expect(retrieveBroker(ctx, key).Name).To(Equal(s.Name))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.MQTT.Brokers.NewWriter(tx).Delete(ctx, key)
			})).To(Succeed())
		})

		It("Should not echo a status the client sent", func(ctx SpecContext) {
			grantWrites(ctx, svc.Rack.EmbeddedKey)
			s := newBroker()
			s.Status = &broker.Status{Message: "Broker connected"}
			res := MustSucceed(create(
				testutil.AuthedCtx(ctx, u),
				apibroker.CreateRequest{Brokers: []broker.Broker{s}},
			))
			Expect(res.Brokers[0].Status).To(BeNil())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.MQTT.Brokers.NewWriter(tx).Delete(ctx, s.Key)
			})).To(Succeed())
		})

		It("Should reject the request without update access on the rack", func(
			ctx SpecContext,
		) {
			grant(ctx, u, allWrites, brokerType, statusType)
			s := newBroker()
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apibroker.CreateRequest{Brokers: []broker.Broker{s}},
			)).Error().To(MatchError(access.ErrDenied))
		})

		It("Should reject the request when create is not granted", func(
			ctx SpecContext,
		) {
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apibroker.CreateRequest{Brokers: []broker.Broker{newBroker()}},
			)).Error().To(MatchError(access.ErrDenied))
		})

		It("Should reject an invalid broker", func(ctx SpecContext) {
			grantWrites(ctx, svc.Rack.EmbeddedKey)
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apibroker.CreateRequest{Brokers: []broker.Broker{
					newBroker(
						func(s *broker.Broker) { s.Host = "" },
					),
				}},
			)).Error().To(MatchError(validate.ErrValidation))
		})
	})

	Describe("Update", func() {
		rename := func(s broker.Broker) apibroker.UpdateRequest {
			return apibroker.UpdateRequest{
				Key: s.Key,
				Update: broker.Update{Variant: broker.IdentityUpdate{
					Identity: broker.Identity{
						Rack:      s.Rack,
						Name:      "Renamed",
						Host:      s.Host,
						Port:      s.Port,
						Transport: s.Transport,
						Auth:      s.Auth,
						ClientID:  s.ClientID,
						KeepAlive: s.KeepAlive,
						Sparkplug: s.Sparkplug,
					},
				}},
			}
		}

		It("Should write only the identity group", func(ctx SpecContext) {
			s := createBroker(ctx, func(s *broker.Broker) {
				s.Reads = []broker.Read{{Variant: broker.SparkplugRead{
					Group:    "Plant",
					EdgeNode: "Line1",
					Tag:      "Pressure",
					Index:    1,
					Channel:  2,
				}}}
			})
			grantWrites(ctx, s.Rack)
			MustSucceed(update(testutil.AuthedCtx(ctx, u), rename(s)))
			got := retrieveBroker(ctx, s.Key)
			Expect(got.Name).To(Equal("Renamed"))
			Expect(got.Reads).To(Equal(s.Reads))
		})

		It("Should write only the configuration group", func(ctx SpecContext) {
			s := createBroker(ctx)
			grantWrites(ctx, s.Rack)
			MustSucceed(update(
				testutil.AuthedCtx(ctx, u),
				apibroker.UpdateRequest{
					Key: s.Key,
					Update: broker.Update{Variant: broker.ConfigurationUpdate{
						Configuration: broker.Configuration{
							Writes: []broker.Write{{Variant: broker.SparkplugWrite{
								Group:    "Plant",
								EdgeNode: "Line1",
								Tag:      "Speed",
								Channel:  2,
							}}},
						},
					}},
				},
			))
			got := retrieveBroker(ctx, s.Key)
			Expect(got.Writes).To(HaveLen(1))
			Expect(got.Writes[0].Variant.(broker.SparkplugWrite).Channel).
				To(Equal(channel.Key(2)))
			Expect(got.Name).To(Equal(s.Name))
		})

		It("Should move a broker only with update access on the new rack", func(
			ctx SpecContext,
		) {
			other := rack.Rack{
				Name:         "Other",
				Integrations: []string{broker.Integration},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Rack.NewWriter(tx).Create(ctx, &other)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.Rack.NewWriter(tx).Delete(ctx, nil, other.Key)
				})).To(Succeed())
			})
			s := createBroker(ctx)
			move := apibroker.UpdateRequest{
				Key: s.Key,
				Update: broker.Update{Variant: broker.IdentityUpdate{
					Identity: broker.Identity{
						Rack:      other.Key,
						Name:      s.Name,
						Host:      s.Host,
						Port:      s.Port,
						Transport: s.Transport,
						Auth:      s.Auth,
						ClientID:  s.ClientID,
						KeepAlive: s.KeepAlive,
						Sparkplug: s.Sparkplug,
					},
				}},
			}
			grantWrites(ctx, s.Rack)
			Expect(update(testutil.AuthedCtx(ctx, u), move)).
				Error().To(MatchError(access.ErrDenied))
			grant(ctx, u, []access.Action{access.ActionUpdate}, other.ResourceID())
			MustSucceed(update(testutil.AuthedCtx(ctx, u), move))
			Expect(retrieveBroker(ctx, s.Key).Rack).To(Equal(other.Key))
		})

		It("Should reject a request with no group", func(ctx SpecContext) {
			s := createBroker(ctx)
			grantWrites(ctx, s.Rack)
			Expect(update(
				testutil.AuthedCtx(ctx, u),
				apibroker.UpdateRequest{Key: s.Key},
			)).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should reject the request when update is not granted", func(
			ctx SpecContext,
		) {
			s := createBroker(ctx)
			Expect(update(testutil.AuthedCtx(ctx, u), rename(s))).
				Error().To(MatchError(access.ErrDenied))
			Expect(retrieveBroker(ctx, s.Key).Name).To(Equal(s.Name))
		})
	})

	Describe("Delete", func() {
		It("Should remove the broker when delete is granted", func(ctx SpecContext) {
			s := createBroker(ctx)
			grantWrites(ctx, s.Rack)
			MustSucceed(del(
				testutil.AuthedCtx(ctx, u),
				apibroker.DeleteRequest{Keys: []broker.Key{s.Key}},
			))
			Expect(svc.MQTT.Brokers.NewRetrieve().
				Where(broker.MatchKeys(s.Key)).
				Exists(ctx, db)).To(BeFalse())
		})

		It("Should reject the request when delete is not granted", func(
			ctx SpecContext,
		) {
			s := createBroker(ctx)
			Expect(del(
				testutil.AuthedCtx(ctx, u),
				apibroker.DeleteRequest{Keys: []broker.Key{s.Key}},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrieveBroker(ctx, s.Key).Key).To(Equal(s.Key))
		})
	})
})
