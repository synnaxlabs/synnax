package framer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	synnax "github.com/synnaxlabs/client"
	"github.com/synnaxlabs/client/channel"
	"github.com/synnaxlabs/client/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Iterator", Ordered, func() {
	var client *synnax.Synnax
	BeforeAll(func() {
		client = synnax.Open(ctx, connParams)
	})
	It("Should correctly iterate through data", func() {
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
		Expect(client.Channels.CreateOne(ctx, &indexCh)).To(Succeed())
		dataCh.Index = indexCh.Key
		Expect(client.Channels.CreateOne(ctx, &dataCh)).To(Succeed())

		w := MustSucceed(client.OpenWriter(ctx, framer.WriterConfig{
			Keys:  []channel.Key{indexCh.Key, dataCh.Key},
			Start: 1 * telem.SecondTS,
		}))
		s1 := telem.NewSecondsTSV(1, 2, 3)
		s2 := telem.NewSeriesV[float64](1.0, 2.0, 3.0)
		Expect(w.Write(ctx, core.Frame{
			Keys:   []channel.Key{indexCh.Key, dataCh.Key},
			Series: []telem.Series{s1, s2},
		}))
		Expect(w.Commit(ctx)).To(BeTrue())
		Expect(w.Close(ctx)).To(Succeed())

		i := MustSucceed(client.OpenIterator(ctx, framer.IteratorConfig{
			Keys:   []channel.Key{indexCh.Key, dataCh.Key},
			Bounds: telem.TimeRangeMax,
		}))
		Expect(i.SeekFirst(ctx)).To(BeTrue())
		Expect(i.Next(ctx, framer.AutoSpan)).To(BeTrue())
		v := i.Value()
		Expect(v.Series).To(HaveLen(2))
		Expect(v.Keys).To(HaveLen(2))
		Expect(i.Next(ctx, framer.AutoSpan)).To(BeFalse())
	})

})
