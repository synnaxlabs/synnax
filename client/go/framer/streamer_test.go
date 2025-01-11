package framer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	synnax "github.com/synnaxlabs/client"
	"github.com/synnaxlabs/client/channel"
	"github.com/synnaxlabs/client/internal/testutil"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"

	"github.com/synnaxlabs/client/framer"
)

var _ = Describe("Streamer", Ordered, func() {
	var client *synnax.Synnax
	BeforeAll(func() {
		client = MustSucceed(synnax.Open(testutil.Config))
	})
	It("Should correctly stream real-time data", func() {
		indexCh := channel.Channel{
			Name:     "index",
			DataType: telem.TimeStampT,
			IsIndex:  true,
		}
		dataCh := channel.Channel{
			Name:     "data",
			DataType: telem.Float64T,
			IsIndex:  false,
		}
		Expect(client.Channels.Create(ctx, &indexCh)).To(Succeed())
		dataCh.Index = indexCh.Key
		Expect(client.Channels.Create(ctx, &dataCh)).To(Succeed())
		s := MustSucceed(client.OpenStreamer(ctx, framer.StreamerConfig{
			Keys: []channel.Key{indexCh.Key, dataCh.Key},
		}))
		w := MustSucceed(client.OpenWriter(ctx, framer.WriterConfig{
			Keys:  []channel.Key{indexCh.Key, dataCh.Key},
			Start: 1 * telem.SecondTS,
			Mode:  cesium.WriterStreamOnly,
		}))
		s1 := telem.NewSeriesV[int64](int64(telem.Now()))
		s2 := telem.NewSeriesV[float64](1.0)
		Expect(w.Write(ctx, core.Frame{
			Keys:   []channel.Key{indexCh.Key, dataCh.Key},
			Series: []telem.Series{s1, s2},
		})).To(BeTrue())
		f := MustSucceed(s.Read())
		Expect(f.Keys).To(ConsistOf(indexCh.Key, dataCh.Key))
		Expect(f.Series).To(HaveLen(2))
		Expect(s.Close()).To(Succeed())
		Expect(w.Close(ctx)).To(Succeed())
	})

})
