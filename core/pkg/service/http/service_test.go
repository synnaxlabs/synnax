// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/http"
	"github.com/synnaxlabs/synnax/pkg/service/http/versions/legacy"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	var (
		svc   *http.Service
		tasks *tasktestutil.Tasks
	)
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(http.OpenService(ctx, http.ServiceConfig{
			DB:      db,
			Channel: channelSvc,
		}))
		tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
			DB:     db,
			Status: statusSvc,
			Stores: svc.Stores(),
		}))
	})

	Describe("OpenService", func() {
		It("Should reject a config missing the DB", func(ctx SpecContext) {
			Expect(http.OpenService(ctx, http.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})

		It("Should reject a config missing the channel service", func(ctx SpecContext) {
			Expect(http.OpenService(ctx, http.ServiceConfig{DB: db})).Error().
				To(MatchError(ContainSubstring("channel: must be non-nil")))
		})
	})

	Describe("Stores", func() {
		It("Should expose one store per HTTP task type", func() {
			types := []string{}
			for _, s := range svc.Stores() {
				types = append(types, s.Type())
			}
			Expect(types).To(ConsistOf(
				"http_read",
				"http_write",
				"http_scan",
			))
		})
	})

	Describe("Normalize", func() {
		DescribeTable(
			"Should import both read shapes released Cores stamped at version 2",
			func(ctx SpecContext, data msgpack.EncodedJSON, nanFill bool) {
				dev := uuid.New().String()
				Expect(tasks.CreateDevice(ctx, dev)).To(Succeed())
				data["device"] = dev
				out := MustSucceed(svc.Read.Normalize(ctx, db, 2, data))
				Expect(out).To(HaveKeyWithValue("nan_fill_enabled", nanFill))
				Expect(out).To(HaveKeyWithValue("device", dev))
			},
			Entry("v2", msgpack.EncodedJSON{}, false),
			Entry("v3", msgpack.EncodedJSON{"nan_fill_enabled": true}, true),
		)

		It(
			"Should export read configs above every version released Cores stamped",
			func() {
				Expect(svc.Read.Version()).To(BeNumerically(">", legacy.LastVersion))
			},
		)
	})

	Describe("Write", func() {
		It("Should store a decoded read config under the given key", func(
			ctx SpecContext,
		) {
			Expect(tasks.CreateDevice(ctx, "dev-1")).To(Succeed())
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
				"device": "dev-1",
				"rate":   25,
				"endpoints": []any{map[string]any{
					"key":    "ep-1",
					"method": "GET",
					"path":   "/telemetry",
				}},
			}))
			data := MustSucceed(svc.Read.Read(ctx, db, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["device"]).To(Equal("dev-1"))
			Expect(data["rate"]).To(BeNumerically("==", 25))
		})

		It("Should apply read config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{}))
			data := MustSucceed(svc.Read.Read(ctx, db, key))
			Expect(data["rate"]).To(BeNumerically("==", 1))
		})

		It("Should apply write endpoint schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Write, msgpack.EncodedJSON{
				"endpoints": []any{map[string]any{"key": "ep-1"}},
			}))
			data := MustSucceed(svc.Write.Read(ctx, db, key))
			Expect(data["endpoints"]).To(HaveExactElements(
				HaveKeyWithValue("method", "POST"),
			))
		})

		It("Should apply scan config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Scan, msgpack.EncodedJSON{}))
			data := MustSucceed(svc.Scan.Read(ctx, db, key))
			Expect(data["rate"]).To(BeNumerically("==", 0.2))
		})

		It("Should return the read validation error for an invalid endpoint", func(
			ctx SpecContext,
		) {
			Expect(svc.Read.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"endpoints": []any{map[string]any{"method": "BOGUS"}},
			})).Error().To(MatchError(ContainSubstring("invalid method: BOGUS")))
		})

		It("Should return the write validation error for an invalid endpoint", func(
			ctx SpecContext,
		) {
			Expect(svc.Write.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"endpoints": []any{map[string]any{"method": "BOGUS"}},
			})).Error().To(MatchError(ContainSubstring("invalid method: BOGUS")))
		})
	})

	DescribeTable("Should reject a scan config whose rate is not positive",
		func(ctx SpecContext, rate float64) {
			Expect(svc.Scan.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"rate": rate,
			})).Error().To(MatchError(ContainSubstring("rate: must be greater than 0")))
		},
		Entry("zero", 0.0),
		Entry("negative", -1.0),
	)

	DescribeTable("Should reject a read config whose rate is not positive",
		func(ctx SpecContext, rate float64) {
			Expect(svc.Read.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"rate": rate,
			})).Error().To(MatchError(ContainSubstring("rate: must be greater than 0")))
		},
		Entry("zero", 0.0),
		Entry("negative", -1.0),
	)
})
