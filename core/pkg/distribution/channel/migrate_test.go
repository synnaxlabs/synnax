package channel_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	tmock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migrate", func() {
	It("should migrate channel names", func() {
		ctx := context.Background()
		node := mock.ProvisionCluster(ctx, 1).Nodes[1]
		channelNet := tmock.NewChannelNetwork()
		cfg := channel.Config{
			HostResolver:     node.Cluster,
			ClusterDB:        node.DB,
			TSChannel:        node.Storage.TS,
			IntOverflowCheck: channel.FixedOverflowChecker(100),
			Transport:        channelNet.New(address.Rand()),
		}
		svc := MustSucceed(channel.OpenService(ctx, cfg, channel.Config{
			ValidateNames: config.False(),
		}))
		originalChannels := []channel.Channel{
			{
				Name:        "test",
				Leaseholder: 1,
				DataType:    telem.TimeStampT,
				Virtual:     true,
			},
			{
				Name:        "name with spaces",
				Leaseholder: 1,
				DataType:    telem.TimeStampT,
				Virtual:     true,
			},
			{
				Name:        "test",
				Leaseholder: 1,
				DataType:    telem.TimeStampT,
				Virtual:     true,
			},
			{
				Name:        "return",
				Leaseholder: 1,
				DataType:    telem.TimeStampT,
				Virtual:     true,
			},
		}
		Expect(svc.CreateMany(ctx, &originalChannels)).To(Succeed())
		svc = MustSucceed(channel.OpenService(ctx, cfg, channel.Config{
			ForceMigration: config.True(),
		}))
		var resChannels []channel.Channel
		Expect(
			svc.
				NewRetrieve().
				WhereKeys(channel.KeysFromChannels(originalChannels)...).
				Entries(&resChannels).
				Exec(ctx, nil),
		).To(Succeed())
		resultingNames := lo.Map(
			resChannels,
			func(ch channel.Channel, _ int) string { return ch.Name },
		)
		Expect(resultingNames).To(ConsistOf(
			"test", "name_with_spaces", "test_1", "return_channel",
		))
	})
})
