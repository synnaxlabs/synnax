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
	Describe("New", func() {
		It("Should correctly attach tracing meta data", func() {
			clientIns := Instrumentation("falamos", InstrumentationConfig{Trace: config.True()})
			clientMw := MustSucceed(falamos.Middleware(falamos.Config{
				Instrumentation: clientIns,
			}))
			oCtx := MustSucceed(clientMw.Exec(
				freighter.Context{
					Context:  ctx,
					Location: freighter.ClientSide,
					Params:   make(freighter.Params),
				},
				freighter.NopFinalizer,
			))
			_, ok := oCtx.Params.Get("traceparent")
			Expect(ok).To(BeTrue())

			serverIns := Instrumentation("falamos", InstrumentationConfig{Trace: config.True()})
			serverMw := MustSucceed(falamos.Middleware(falamos.Config{
				Instrumentation: serverIns,
			}))
			oCtx = MustSucceed(serverMw.Exec(
				freighter.Context{
					Context:  ctx,
					Location: freighter.ServerSide,
					Params:   oCtx.Params,
				},
				freighter.NopFinalizer,
			))
		})
	})
})
