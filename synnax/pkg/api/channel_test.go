package api_test

import (
	"context"

	roacherrors "github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/synnax/pkg/api/mock"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("ChannelReader", Ordered, func() {
	var (
		builder *mock.Builder
		prov    api.Provider
		svc     *api.ChannelService
	)
	BeforeAll(func() {
		builder = mock.New()
		prov = builder.New()
		svc = api.NewChannelService(prov)
	})
	AfterAll(func() {
		Expect(builder.Close()).To(Succeed())
		Expect(builder.Cleanup()).To(Succeed())
	})
	Describe("Create", func() {
		It("Should create a new Channel", func() {
			res, err := svc.Create(context.TODO(), api.ChannelCreateRequest{
				Channel: api.Channel{
					Name:     "test",
					NodeID:   1,
					DataType: telem.Float64T,
					Rate:     25 * telem.Hz,
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
			Expect(err).To(HaveOccurred())
			Expect(err.Err).To(HaveOccurred())
			flds, ok := err.Err.(errors.Fields)
			Expect(ok).To(BeTrue())
			Expect(flds[0].Field).To(Equal(field))
			Expect(flds[0].Message).To(Equal(message))
			Expect(len(res.Channels)).To(Equal(0))
		},
			Entry("No Array Type", api.Channel{
				Name:   "test",
				NodeID: 1,
				Rate:   25 * telem.Hz,
			}, "Channel.data_type", "required"),
			Entry("No Array Rate", api.Channel{
				Name:     "test",
				NodeID:   1,
				DataType: telem.Float64T,
			}, "Channel.rate", "required"),
		)
	})
	Describe("RetrieveP", func() {
		It("Should retrieve all created channels", func() {
			res, err := svc.Retrieve(context.TODO(), api.ChannelRetrieveRequest{})
			Expect(err).To(Equal(errors.Nil))
			Expect(res.Channels).To(HaveLen(1))
		})
		It("Should retrieve a Channel by its key", func() {
			res, err := svc.Retrieve(context.TODO(), api.ChannelRetrieveRequest{
				Keys: []string{"1-1"},
			})
			Expect(err).To(Equal(errors.Nil))
			Expect(res.Channels).To(HaveLen(1))
		})
		It("Should return an error if the key can't be parsed", func() {
			res, err := svc.Retrieve(context.TODO(), api.ChannelRetrieveRequest{
				Keys: []string{"1-1-1"},
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
		It("Should retrieve channels by their name", func() {
			res, err := svc.Retrieve(context.TODO(), api.ChannelRetrieveRequest{
				Names: []string{"test"},
			})
			Expect(err).To(Equal(errors.Nil))
			Expect(res.Channels).To(HaveLen(1))
			for _, ch := range res.Channels {
				Expect(ch.Name).To(Equal("test"))
			}
		})
	})
})
