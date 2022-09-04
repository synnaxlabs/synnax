package channel_test

import (
	"context"
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/delta/pkg/distribution"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/delta/pkg/distribution/core/mock"
	"github.com/arya-analytics/delta/pkg/storage"
	"github.com/arya-analytics/freighter/fmock"
	"github.com/arya-analytics/x/config"
	"go.uber.org/zap"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	ctx = context.Background()
)

func TestChannel(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "channel Suite")
}

func provisionServices(logger *zap.Logger) (*mock.CoreBuilder, map[core.NodeID]*channel.Service) {
	services := make(map[aspen.NodeID]*channel.Service)
	net := fmock.NewNetwork[channel.CreateMessage, channel.CreateMessage]()
	builder := mock.NewCoreBuilder(distribution.Config{
		Logger:  logger,
		Storage: storage.Config{MemBacked: config.BoolPointer(true)},
	})
	core1 := builder.New()
	services[1] = channel.New(
		core1.Cluster,
		core1.Storage.Gorpify(),
		core1.Storage.TS,
		net.RouteUnary(core1.Config.AdvertiseAddress),
	)
	core2 := builder.New()
	services[2] = channel.New(
		core2.Cluster,
		core2.Storage.Gorpify(),
		core2.Storage.TS,
		net.RouteUnary(core2.Config.AdvertiseAddress),
	)
	Eventually(func(g Gomega) {
		g.Expect(core1.Cluster.Nodes()).To(HaveLen(2))
	}).Should(Succeed())
	return builder, services

}
