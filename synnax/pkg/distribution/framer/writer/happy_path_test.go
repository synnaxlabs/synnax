package writer_test

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"io"
)

var (
	channels = []channel.Channel{
		{
			Name:     "test1",
			Rate:     1 * telem.Hz,
			DataType: telem.Int64T,
		},
		{
			Name:     "test2",
			Rate:     1 * telem.Hz,
			DataType: telem.Int64T,
		},
		{
			Name:     "test3",
			Rate:     1 * telem.Hz,
			DataType: telem.Int64T,
		},
	}
)

var _ = Describe("Happy Path", func() {
	scenarios := [](func() scenario){
		gatewayOnlyScenario,
		peerOnlyScenario,
		mixedScenario,
	}
	for _, sF := range scenarios {
		var s scenario
		BeforeEach(func() {
			s = sF()
		})
		AfterEach(func() { Expect(s.close.Close()).To(Succeed()) })
		Specify("It should write data to the channels", func() {
			writer := MustSucceed(s.service.New(context.TODO(), writer.Config{Keys: s.keys, Start: 10 * telem.SecondTS}))
			Expect(writer.Write(core.NewFrame(
				s.keys,
				[]telem.Array{
					telem.NewArrayV[int64](1, 2, 3),
					telem.NewArrayV[int64](3, 4, 5),
					telem.NewArrayV[int64](5, 6, 7),
				},
			))).To(BeTrue())
			Expect(writer.Commit()).To(BeTrue())
			Expect(writer.Write(
				core.NewFrame(
					s.keys,
					[]telem.Array{
						telem.NewArrayV[int64](1, 2, 3),
						telem.NewArrayV[int64](3, 4, 5),
						telem.NewArrayV[int64](5, 6, 7),
					},
				),
			)).To(BeTrue())
			Expect(writer.Commit()).To(BeTrue())
			Expect(writer.Close()).To(Succeed())
		})
	}
})

type scenario struct {
	keys    channel.Keys
	service *writer.Service
	close   io.Closer
}

func gatewayOnlyScenario() scenario {
	builder, services := provision(1, zap.NewNop())

	svc := services[1]

	Expect(svc.channel.CreateMany(&channels)).To(Succeed())
	keys := channel.KeysFromChannels(channels)
	return scenario{
		keys:    keys,
		service: svc.writer,
		close:   builder,
	}
}

func peerOnlyScenario() scenario {
	builder, services := provision(4, zap.NewNop())

	svc := services[1]

	for i, ch := range channels {
		ch.NodeID = dcore.NodeID(i + 2)
		Expect(svc.channel.Create(&ch)).To(Succeed())
	}

	Expect(svc.channel.CreateMany(&channels)).To(Succeed())
	keys := channel.KeysFromChannels(channels)
	return scenario{
		keys:    keys,
		service: svc.writer,
		close:   builder,
	}
}

func mixedScenario() scenario {
	builder, services := provision(3, zap.NewNop())

	svc := services[1]

	for i, ch := range channels {
		ch.NodeID = dcore.NodeID(i + 1)
		Expect(svc.channel.Create(&ch)).To(Succeed())
	}

	Expect(svc.channel.CreateMany(&channels)).To(Succeed())
	keys := channel.KeysFromChannels(channels)
	return scenario{
		keys:    keys,
		service: svc.writer,
		close:   builder,
	}
}
