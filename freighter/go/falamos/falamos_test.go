// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package falamos_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/falamos"
	"github.com/synnaxlabs/x/config"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Falamos", func() {
	Describe("Name", func() {
		It("Should correctly attach tracing metadata", func() {
			clientIns := Instrumentation("falamos", InstrumentationConfig{Trace: config.True()})
			clientMw := MustSucceed(falamos.Middleware(falamos.Config{
				Instrumentation: clientIns,
			}))
			oCtx := MustSucceed(clientMw.Exec(
				freighter.Context{
					Context: ctx,
					Role:    freighter.Client,
					Params:  make(freighter.Params),
				},
				freighter.NopFinalizer,
			))
			_, ok := oCtx.Get("alamos-traceparent")
			Expect(ok).To(BeTrue())

			serverIns := Instrumentation("falamos", InstrumentationConfig{Trace: config.True()})
			serverMw := MustSucceed(falamos.Middleware(falamos.Config{
				Instrumentation: serverIns,
			}))
			oCtx = MustSucceed(serverMw.Exec(
				freighter.Context{
					Context: ctx,
					Role:    freighter.Server,
					Params:  oCtx.Params,
				},
				freighter.NopFinalizer,
			))
			_, ok = oCtx.Get("alamos-traceparent")
			Expect(ok).To(BeTrue())
		})
	})
})
