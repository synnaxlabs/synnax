package runtime_test

import (
	"math/rand"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer/text"
	"github.com/synnaxlabs/arc/compiler"
	"github.com/synnaxlabs/arc/module"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Runtime", Ordered, func() {
	var (
		c    *calculation.Service
		dist mock.Node
	)

	BeforeAll(func() {
		distB := mock.NewCluster()
		dist = distB.Provision(ctx)
		c = MustSucceed(calculation.OpenService(ctx, calculation.ServiceConfig{
			Framer:            dist.Framer,
			Channel:           dist.Channel,
			ChannelObservable: dist.Channel.NewObservable(),
		}))
	})

	AfterAll(func() {
		Expect(c.Close()).To(Succeed())
	})

	It("Should run a basic test", func() {
		ch := &channel.Channel{
			Name:     "ox_pt_1",
			Virtual:  true,
			DataType: telem.Float32T,
		}
		Expect(dist.Channel.Create(ctx, ch)).To(Succeed())

		source := `ox_pt_1 > f32(0.5) -> print{message: "dog"}`

		printType := types.NewTask()
		printType.Config.Put("message", types.String{})

		resolver := symbol.MapResolver{
			"ox_pt_1": symbol.Symbol{
				Name: "ox_pt_1",
				Kind: symbol.KindChannel,
				Type: types.Chan{ValueType: types.F32{}},
				ID:   int(ch.Key()),
			},
			"print": symbol.Symbol{
				Name: "print",
				Kind: symbol.KindTask,
				Type: printType,
			},
		}

		prog := MustSucceed(text.Parse(source))
		analysis := text.Analyze(prog, text.Options{Resolver: resolver})
		Expect(analysis.Ok()).To(BeTrue(), analysis.String())
		wasmBytes := MustSucceed(compiler.Compile(compiler.Config{
			Program:  prog,
			Analysis: &analysis,
		}))
		mod := MustSucceed(module.Assemble(prog, analysis, wasmBytes))
		MustSucceed(runtime.New(ctx, runtime.Config{
			Channel: dist.Channel,
			Framer:  dist.Framer,
			Module:  mod,
		}))

		writer := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
			Keys:  []channel.Key{ch.Key()},
			Start: telem.Now(),
		}))
		for {
			Expect(writer.Write(core.UnaryFrame(ch.Key(), telem.NewSeriesV[float32](rand.Float32())))).To(BeTrue())
		}
	})
})
