package api_test

import (
	"context"
	"github.com/arya-analytics/delta/pkg/api"
	"github.com/arya-analytics/delta/pkg/api/errors"
	"github.com/arya-analytics/delta/pkg/api/mock"
	"github.com/arya-analytics/x/telem"
	roacherrors "github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Channel", Ordered, func() {
	var (
		builder *mock.ProviderBuilder
		prov    api.Provider
		svc     *api.ChannelService
	)
	BeforeAll(func() {
		builder = mock.NewProviderBuilder()
		prov = builder.New()
		svc = api.NewChannelService(prov)
	})
	AfterAll(func() {
		Expect(builder.Close()).To(Succeed())
		Expect(builder.Cleanup()).To(Succeed())
	})
	Describe("Create", func() {
		It("Should create a new channel", func() {
			res, err := svc.Create(context.TODO(), api.ChannelCreateRequest{
				Channel: api.Channel{
					Name:    "test",
					NodeID:  1,
					Density: telem.Float64,
					Rate:    25 * telem.Hz,
				},
			})
			Expect(err).To(Equal(errors.Nil))
			Expect(res.Channels).To(HaveLen(1))
		})
		DescribeTable("Validation Errors", func(
			ch api.Channel,
			field string,
			message string,
		) {
			res, err := svc.Create(context.TODO(), api.ChannelCreateRequest{
				Channel: ch,
			})
			flds, ok := err.Err.(errors.Fields)
			Expect(ok).To(BeTrue())
			Expect(flds[0].Field).To(Equal(field))
			Expect(flds[0].Message).To(Equal(message))
			Expect(len(res.Channels)).To(Equal(0))
		},
			Entry("No node id", api.Channel{
				Name:    "test",
				Density: telem.Float64,
				Rate:    25 * telem.Hz,
			}, "channel.node_id", "required"),
			Entry("No Data Type", api.Channel{
				Name:   "test",
				NodeID: 1,
				Rate:   25 * telem.Hz,
			}, "channel.data_type", "required"),
			Entry("No Data Rate", api.Channel{
				Name:    "test",
				NodeID:  1,
				Density: telem.Float64,
			}, "channel.data_rate", "required"),
		)
	})
	Describe("Retrieve", func() {
		Context("All", func() {
			It("Should retrieve all created channels", func() {
				res, err := svc.Retrieve(context.TODO(), api.ChannelRetrieveRequest{})
				Expect(err).To(Equal(errors.Nil))
				Expect(res.Channels).To(HaveLen(1))
			})
			It("Should retrieve a channel by its key", func() {
				res, err := svc.Retrieve(context.TODO(), api.ChannelRetrieveRequest{
					Key: []string{"1-1"},
				})
				Expect(err).To(Equal(errors.Nil))
				Expect(res.Channels).To(HaveLen(1))
			})
			It("Should return an error if the key can't be parsed", func() {
				res, err := svc.Retrieve(context.TODO(), api.ChannelRetrieveRequest{
					Key: []string{"1-1-1"},
				})
				Expect(err).To(Equal(errors.Parse(roacherrors.New("[channel] - invalid key format"))))
				Expect(res.Channels).To(HaveLen(0))
			})
			It("Should retrieve channels by their node ID", func() {
				res, err := svc.Retrieve(context.TODO(), api.ChannelRetrieveRequest{
					NodeID: 1,
				})
				Expect(err).To(Equal(errors.Nil))
				Expect(res.Channels).To(HaveLen(1))
			})
		})
	})
})
