package kv_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/kv"
	kvx "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
)

var _ = Describe("Channel", func() {
	var (
		db  kvx.DB
		svc *kv.ChannelService
	)
	BeforeEach(func() {
		db = memkv.New()
		svc = kv.NewChannelService(db)
	})
	AfterEach(func() {
		Expect(db.Close()).To(Succeed())
	})
	Describe("SetMultiple", func() {
		It("Should set a channel in the DB", func() {
			Expect(svc.Set(channel.Channel{Key: 1})).To(Succeed())
		})
	})

	Describe("Get", func() {
		It("Should get a channel from the DB", func() {
			Expect(svc.Set(channel.Channel{Key: 1})).To(Succeed())
			chs, err := svc.Get(1)
			Expect(err).To(Succeed())
			Expect(chs).To(HaveLen(1))
			Expect(chs[0].Key).To(Equal(channel.Key(1)))
		})
	})
	Describe("Exists", func() {
		It("Should return true if the channel exists", func() {
			Expect(svc.Set(channel.Channel{Key: 1})).To(Succeed())
			exists, err := svc.Exists(1)
			Expect(err).To(Succeed())
			Expect(exists).To(BeTrue())
		})
		It("Should return false if the channel does not exist", func() {
			exists, err := svc.Exists(1)
			Expect(err).To(Succeed())
			Expect(exists).To(BeFalse())
		})
	})
})
