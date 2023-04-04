package falamos_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/alamos/falamos"
	"github.com/synnaxlabs/freighter"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Falamos", func() {
	Describe("InstrumentationMiddleware", func() {
		It("Should correctly attach tracing meta data", func() {
			clientCtx, _ := alamos.Trace(
				alamos.Dev("falamos", false, "falamos.test.client"),
				"falamos.test.client",
			)
			clientMw := falamos.InstrumentationMiddleware()
			oCtx := MustSucceed(clientMw.Exec(
				freighter.Context{
					Context:  clientCtx,
					Location: freighter.ClientSide,
					Params:   make(freighter.Params),
				},
				freighter.NopFinalizer,
			))
			_, ok := oCtx.Params.Get("traceparent")
			Expect(ok).To(BeTrue())

			serverCtx := alamos.Dev("falamos", false, "falamos.test.server")
			serverMw := MustSucceed(falamos.InstrumentationMiddleware())
			oCtx = MustSucceed(serverMw.Exec(
				freighter.Context{
					Context:  serverCtx,
					Location: freighter.ServerSide,
					Params:   oCtx.Params,
				},
				freighter.NopFinalizer,
			))
		})
	})
})
