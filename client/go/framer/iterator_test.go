package framer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	synnax "github.com/synnaxlabs/client"
	"github.com/synnaxlabs/client/channel"
	"github.com/synnaxlabs/client/framer"
	"github.com/synnaxlabs/client/internal/testutil"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Iterator", Ordered, func() {
	var client *synnax.Synnax
	BeforeAll(func() {
		client = MustSucceed(synnax.Open(testutil.Config))
	})
	It("Should correctly iterate through data", func() {
		indexCh := channel.Channel{Name: "index", DataType: synnax.TimeStampT, IsIndex: true}
		dataCh := channel.Channel{Name: "data", DataType: synnax.Float64T}
		Expect(client.Channels.Create(ctx, &indexCh)).To(Succeed())
		dataCh.Index = indexCh.Key
		Expect(client.Channels.Create(ctx, &dataCh)).To(Succeed())
		w := MustSucceed(client.OpenWriter(ctx, framer.WriterConfig{
			Keys:  []channel.Key{indexCh.Key, dataCh.Key},
			Start: 1 * synnax.SecondTS,
		}))
		s1 := synnax.NewSecondsTSV(1, 2, 3)
		s2 := synnax.NewSeriesV[float64](1.0, 2.0, 3.0)
		Expect(w.Write(ctx, core.Frame{
			Keys:   []channel.Key{indexCh.Key, dataCh.Key},
			Series: []synnax.Series{s1, s2},
		}))
		Expect(w.Commit(ctx)).To(BeTrue())
		Expect(w.Close(ctx)).To(Succeed())

		i := MustSucceed(client.OpenIterator(ctx, framer.IteratorConfig{
			Keys:   []channel.Key{indexCh.Key, dataCh.Key},
			Bounds: synnax.TimeRangeMax,
		}))
		Expect(i.SeekFirst(ctx)).To(BeTrue())
		Expect(i.Next(ctx, framer.AutoSpan)).To(BeTrue())
		v := i.Value()
		Expect(v.Series).To(HaveLen(2))
		Expect(v.Keys).To(HaveLen(2))
		Expect(i.Next(ctx, framer.AutoSpan)).To(BeFalse())
		Expect(i.Close()).To(Succeed())
	})

})
