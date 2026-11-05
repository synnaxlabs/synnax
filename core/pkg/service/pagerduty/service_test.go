// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pagerduty_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	pd "github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Service", func() {
	var (
		svc   *pd.Service
		tasks *tasktestutil.Tasks
	)
	BeforeEach(func(ctx SpecContext) {
		// The suite's DB holds a task table of its own.
		db := DeferClose(gorp.Wrap(memkv.New()))
		svc = MustOpen(pd.OpenService(ctx, pd.ServiceConfig{
			DB: db,
		}))
		tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
			DB:     db,
			Stores: svc.Stores(),
		}))
	})

	Describe("OpenService", func() {
		It("Should reject a config missing the DB", func(ctx SpecContext) {
			Expect(pd.OpenService(ctx, pd.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})
	})

	Describe("Stores", func() {
		It("Should expose the alert store", func() {
			types := []string{}
			for _, s := range svc.Stores() {
				types = append(types, s.Type())
			}
			Expect(types).To(ConsistOf("pagerduty_alert"))
		})
	})

	Describe("Write", func() {
		It("Should store a decoded alert config under the given key", func(
			ctx SpecContext,
		) {
			Expect(tasks.CreateStatus(ctx, "status-1")).To(Succeed())
			key := MustSucceed(tasks.Create(ctx, svc.Alert, msgpack.EncodedJSON{
				"routing_key": "rk-1",
				"auto_start":  true,
				"alerts": []any{map[string]any{
					"key":    "alert-1",
					"status": "status-1",
				}},
			}))
			data := MustSucceed(svc.Alert.Read(ctx, nil, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["routing_key"]).To(Equal("rk-1"))
			Expect(data["auto_start"]).To(BeTrue())
		})
	})
})
