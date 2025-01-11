package channel_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/client"
	"github.com/synnaxlabs/client/internal/testutil"
	"github.com/synnaxlabs/x/query"
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
			ch := synnax.Channel{
				Name:     "test",
				DataType: synnax.TimeStampT,
				IsIndex:  true,
			}
			Expect(client.Channels.Create(ctx, &ch)).To(Succeed())
			Expect(ch.Key).ToNot(Equal(0))
		})
		It("Should retrieve a channel if its name already exists", func() {
			v := uuid.New()
			ch := synnax.Channel{
				Name:     v.String(),
				DataType: synnax.TimeStampT,
				Virtual:  true,
			}
			Expect(client.Channels.Create(ctx, &ch)).To(Succeed())
			Expect(ch.Key).ToNot(Equal(0))
			ch2 := synnax.Channel{
				Name:     v.String(),
				DataType: synnax.TimeStampT,
				Virtual:  true,
			}
			Expect(client.Channels.Create(ctx, &ch2, channel.RetrieveIfNameExists())).To(Succeed())
			Expect(ch2.Key).To(Equal(ch.Key))
		})
	})
	Describe("Retrieve", func() {
		It("Should correctly retrieve a channel", func() {
			ch := synnax.Channel{
				Name:     "test",
				DataType: synnax.TimeStampT,
				IsIndex:  true,
			}
			Expect(client.Channels.Create(ctx, &ch)).To(Succeed())
			Expect(ch.Key).ToNot(Equal(0))
			resCh, err := client.Channels.Retrieve(ctx, channel.WhereKey(ch.Key))
			Expect(err).ToNot(HaveOccurred())
			Expect(resCh.Key).To(Equal(ch.Key))
		})
	})
	Describe("Delete", func() {
		It("Should correctly delete a channel", func() {
			ch := synnax.Channel{
				Name:     "test",
				DataType: synnax.TimeStampT,
				IsIndex:  true,
			}
			Expect(client.Channels.Create(ctx, &ch)).To(Succeed())
			Expect(ch.Key).ToNot(Equal(0))
			Expect(client.Channels.Delete(ctx, channel.WhereKey(ch.Key)))
			_, err := client.Channels.Retrieve(ctx, channel.WhereKey(ch.Key))
			Expect(err).To(HaveOccurredAs(query.Error))
		})
	})
})
