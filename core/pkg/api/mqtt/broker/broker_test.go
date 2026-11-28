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

var brokerType = resource.ID{Type: resource.TypeMqttBroker}

var _ = Describe("Service", func() {
	var u user.User
	BeforeEach(func(ctx SpecContext) { u = createUser(ctx) })

	newBroker := func(overrides ...func(*broker.Broker)) broker.Broker {
		s := broker.Broker{
			Key:       uuid.New(),
			Rack:      svc.Rack.EmbeddedKey,
			Name:      "API-" + uuid.New().String(),
			Transport: broker.Transport{Variant: broker.TcpTransport{}},
			Host:      "plc.example.com",
			Auth:      broker.Auth{Variant: broker.NoneAuth{}},
			Sparkplug: broker.Sparkplug{
				Mode: broker.SparkplugMode{Variant: broker.PassiveSparkplugMode{}},
			},
		}
		for _, override := range overrides {
			override(&s)
		}
		return s
	}
	grantRetrieve := func(ctx SpecContext) {
		GinkgoHelper()
		grant(ctx, u, []access.Action{access.ActionRetrieve}, brokerType)
	}
	createBroker := func(ctx SpecContext, overrides ...func(*broker.Broker)) broker.Broker {
		GinkgoHelper()
		s := newBroker(overrides...)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return gorp.NewCreate[broker.Key, broker.Broker]().Entry(&s).Exec(ctx, tx)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return gorp.NewDelete[broker.Key, broker.Broker]().
					Where(gorp.MatchKeys[broker.Key, broker.Broker](s.Key)).
					Exec(ctx, tx)
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
})
