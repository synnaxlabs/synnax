package framer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/synnax/pkg/storage/control"
	"github.com/synnaxlabs/synnax/pkg/storage/framer"
	"github.com/synnaxlabs/synnax/pkg/storage/mock"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Writer", Ordered, func() {
	var store *storage.Store
	BeforeAll(func() {
		b := mock.NewBuilder()
		store = b.New()
	})
	AfterAll(func() { Expect(store.Close()).To(Succeed()) })
	Describe("Happy Path", func() {
		It("Should correctly write a frame of telemetry to the DB", func() {
			idx := framer.Channel{Key: 1, DataType: telem.TimeStampT, IsIndex: true}
			data := framer.Channel{Key: 2, DataType: telem.Int64T, Index: idx.Key}
			Expect(store.Framer.CreateChannel(ctx, idx, data)).To(Succeed())
			w := MustSucceed(store.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start:       telem.SecondTS * 5,
				Channels:    []framer.ChannelKey{idx.Key, data.Key},
				Authorities: []control.Authority{1, 1},
			}))
			f := framer.Frame{
				Keys: []framer.ChannelKey{idx.Key, data.Key},
				Series: []telem.Series{
					telem.NewSecondsTSV(5, 6, 7),
					telem.NewArrayV[int64](7, 8, 9),
				},
			}
			Expect(w.Write(ctx, f)).To(BeTrue())
			_, ok := w.Commit(ctx)
			Expect(ok).To(BeTrue())
			Expect(w.Close()).To(Succeed())
		})
	})
})
