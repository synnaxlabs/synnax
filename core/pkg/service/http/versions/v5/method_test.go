// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	v4 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("Method", func() {
	DescribeTable("Should accept a supported method",
		func(m v5.Method) { Expect(m.IsValid()).To(BeTrue()) },
		Entry("QUERY", v5.MethodQuery),
		Entry("GET", v5.MethodGet),
		Entry("PATCH", v5.MethodPatch),
	)

	DescribeTable("Should refuse a removed method",
		func(m v5.Method) { Expect(m.IsValid()).To(BeFalse()) },
		Entry("TRACE", v5.Method("TRACE")),
		Entry("CONNECT", v5.Method("CONNECT")),
	)

	It("Should refuse a read endpoint that uses TRACE", func() {
		Expect(v5.ReadEndpoint{Method: "TRACE"}.Validate()).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("method")),
		))
	})

	It("Should refuse a write endpoint that uses CONNECT", func() {
		Expect(v5.WriteEndpoint{Method: "CONNECT"}.Validate()).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("method")),
		))
	})
})

var _ = Describe("MigrateWriteConfig", func() {
	It("Should keep the endpoints and their methods", func(ctx SpecContext) {
		old := v4.WriteConfig{
			Device: "dev",
			Endpoints: []v4.WriteEndpoint{
				{Key: "ep", Method: v4.MethodPut, Path: "/valve"},
			},
		}
		cfg := MustSucceed(v5.MigrateWriteConfig(ctx, old))
		Expect(cfg.Device).To(Equal("dev"))
		Expect(cfg.Endpoints).To(HaveLen(1))
		Expect(cfg.Endpoints[0].Method).To(Equal(v5.MethodPut))
		Expect(cfg.Endpoints[0].Path).To(Equal("/valve"))
	})

	It("Should keep the method of an endpoint", func(ctx SpecContext) {
		ep := MustSucceed(v5.MigrateWriteEndpoint(ctx, v4.WriteEndpoint{
			Key:    "ep",
			Method: v4.MethodPatch,
		}))
		Expect(ep.Method).To(Equal(v5.MethodPatch))
	})

	It("Should store the shape that v4 stored under the same table name", func() {
		Expect(v5.WriteConfig{}.CustomTypeName()).
			To(Equal(v4.WriteConfig{}.CustomTypeName()))
	})
})

var _ = Describe("Removed method warnings", func() {
	var (
		db   *gorp.DB
		ins  alamos.Instrumentation
		logs *observer.ObservedLogs
	)
	BeforeEach(func() {
		db = DeferClose(gorp.Wrap(memkv.New()))
		var core zapcore.Core
		core, logs = observer.New(zapcore.WarnLevel)
		ins = alamos.New("test", alamos.WithLogger(MustSucceed(alamos.NewLogger(
			alamos.LoggerConfig{ZapLogger: zap.New(core)},
		))))
	})
	warned := func() []any {
		return lo.Map(
			logs.FilterMessage(
				"an HTTP endpoint uses a method that is no longer supported",
			).All(),
			func(e observer.LoggedEntry, _ int) any {
				return e.ContextMap()["endpoint"]
			},
		)
	}

	It("Should warn about each read endpoint that uses TRACE or CONNECT",
		func(ctx SpecContext) {
			cfg := v4.ReadConfig{Key: uuid.New(), Endpoints: []v4.ReadEndpoint{
				{Key: "trace", Method: "TRACE"},
				{Key: "get", Method: v4.MethodGet},
				{Key: "connect", Method: "CONNECT"},
			}}
			Expect(gorp.WrapWriter[uuid.UUID, v4.ReadConfig](db).Set(ctx, cfg)).
				To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				Instrumentation: ins,
				DB:              db,
				Namespace:       "http_read_config",
				Migrations:      []gorp.Migration{v5.ReadMigration},
			})).To(Succeed())
			Expect(warned()).To(ConsistOf("trace", "connect"))
		},
	)

	It("Should warn about each write endpoint that uses TRACE or CONNECT",
		func(ctx SpecContext) {
			cfg := v4.WriteConfig{
				Key: uuid.New(),
				Endpoints: []v4.WriteEndpoint{
					{Key: "post", Method: v4.MethodPost},
					{Key: "trace", Method: "TRACE"},
				},
			}
			Expect(gorp.WrapWriter[uuid.UUID, v4.WriteConfig](db).Set(ctx, cfg)).
				To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				Instrumentation: ins,
				DB:              db,
				Namespace:       "http_write_config",
				Migrations:      []gorp.Migration{v5.WriteMigration},
			})).To(Succeed())
			Expect(warned()).To(ConsistOf("trace"))
		},
	)
})
