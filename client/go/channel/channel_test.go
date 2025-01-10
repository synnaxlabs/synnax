package channel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/client"
	"github.com/synnaxlabs/client/internal/testutil"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"

	"github.com/synnaxlabs/client/channel"
)

var _ = Describe("Channel", Ordered, func() {
	var client *synnax.Synnax
	BeforeAll(func() {
		client = MustSucceed(synnax.Open(testutil.Config))
	})
	Describe("create", func() {
		It("Should correctly create a channel", func() {
			ch := channel.Channel{
				Name:     "test",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}
			Expect(client.Channels.CreateOne(ctx, &ch)).To(Succeed())
			Expect(ch.Key).ToNot(Equal(0))
		})
	})
	Describe("Retrieve", func() {
		It("Should correctly retrieve a channel", func() {
			ch := channel.Channel{
				Name:     "test",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}
			Expect(client.Channels.CreateOne(ctx, &ch)).To(Succeed())
			Expect(ch.Key).ToNot(Equal(0))
			resCh, err := client.Channels.RetrieveOne(ctx, channel.RetrieveRequest{
				Keys: []channel.Key{ch.Key},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resCh.Key).To(Equal(ch.Key))
		})
	})
	Describe("Delete", func() {
		It("Should correctly delete a channel", func() {
			ch := channel.Channel{
				Name:     "test",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}
			Expect(client.Channels.CreateOne(ctx, &ch)).To(Succeed())
			Expect(ch.Key).ToNot(Equal(0))
			Expect(client.Channels.Delete(ctx, channel.DeleteRequest{
				Keys: []channel.Key{ch.Key},
			})).To(Succeed())
			_, err := client.Channels.RetrieveOne(ctx, channel.RetrieveRequest{
				Keys: []channel.Key{ch.Key},
			})
			Expect(err).To(HaveOccurredAs(query.Error))
		})
	})
})
