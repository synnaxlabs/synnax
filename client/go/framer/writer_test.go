package framer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/client"
	"github.com/synnaxlabs/client/channel"
	"github.com/synnaxlabs/client/internal/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Writer", Ordered, func() {
	var client *synnax.Synnax
	BeforeAll(func() {
		client = MustSucceed(synnax.Open(testutil.Config))
	})
	It("Should correctly write a single frame", func() {
		indexCh := channel.Channel{Name: "index", DataType: telem.TimeStampT, IsIndex: true}
		dataCh := channel.Channel{Name: "data", DataType: telem.Float64T, IsIndex: false}
		Expect(client.Channels.Create(ctx, &indexCh)).To(Succeed())
		dataCh.Index = indexCh.Key
		Expect(client.Channels.Create(ctx, &dataCh)).To(Succeed())
		w := MustSucceed(client.OpenWriter(ctx, synnax.WriterConfig{
			Keys:  []channel.Key{indexCh.Key, dataCh.Key},
			Start: telem.Now(),
		}))
		s1 := telem.NewSeriesV[int64](int64(telem.Now()))
		s2 := telem.NewSeriesV[float64](1.0)
		Expect(w.Write(ctx, synnax.Frame{
			Keys:   []channel.Key{indexCh.Key, dataCh.Key},
			Series: []telem.Series{s1, s2},
		})).To(BeTrue())
		Expect(w.Commit(ctx)).To(BeTrue())
		Expect(w.Close(ctx)).To(Succeed())
	})

})
