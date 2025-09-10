package runtime_test

import (
	"math/rand"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/text"
	"github.com/synnaxlabs/arc/compiler"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/module"
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

		printType := ir.NewTask()
		printType.Config.Put("message", ir.String{})

		resolver := ir.MapResolver{
			"ox_pt_1": ir.Symbol{
				Name: "ox_pt_1",
				Kind: ir.KindChannel,
				Type: ir.Chan{ValueType: ir.F32{}},
				ID:   int(ch.Key()),
			},
			"print": ir.Symbol{
				Name: "print",
				Kind: ir.KindStage,
				Type: printType,
			},
		}

		prog := MustSucceed(text.Parse(source))
		analysis := analyzer.AnalyzeProgram(prog, text.Options{Resolver: resolver})
		Expect(analysis.Ok()).To(BeTrue(), analysis.String())
		wasmBytes := MustSucceed(compiler.Compile(compiler.Config{
			Program:  prog,
			Analysis: &analysis,
		}))
		mod := MustSucceed(module.Assemble(prog, analysis, wasmBytes))
		MustSucceed(runtime.Open(ctx, runtime.Config{
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
