package api_test

import (
	"context"
	"github.com/arya-analytics/client-go/segment"
	"github.com/arya-analytics/delta/pkg/api"
	"github.com/arya-analytics/delta/pkg/api/errors"
	"github.com/arya-analytics/delta/pkg/api/mock"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/freighter/fmock"
	"github.com/arya-analytics/x/telem"
	roacherrors "github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gleak"
	"time"
)

var _ = Describe("SegmentWriter", Ordered, func() {
	var (
		transport *fmock.StreamTransport[api.WriterRequest, api.WriterResponse]
		builder   *mock.ProviderBuilder
		prov      api.Provider
		svc       *api.SegmentService
		keys      channel.Keys
	)
	BeforeAll(func() {
		builder = mock.NewProviderBuilder()
		prov = builder.New()
		svc = api.NewSegmentService(prov)
		ch, err := prov.Config.Channel.NewCreate().
			WithName("test").
			WithRate(25*telem.Hz).
			WithDataType(telem.Float64).
			WithNodeID(1).
			ExecN(ctx, 2)
		Expect(err).To(BeNil())
		keys = channel.KeysFromChannels(ch)
	})
	AfterAll(func() {
		Expect(builder.Close()).To(Succeed())
		Expect(builder.Cleanup()).To(Succeed())
	})
	BeforeEach(func() {
		routines := gleak.Goroutines()
		DeferCleanup(func() {
			Eventually(gleak.Goroutines).WithTimeout(time.Second).ShouldNot(gleak.HaveLeaked(routines))
		})
		transport = fmock.NewStreamTransport[api.WriterRequest, api.WriterResponse](1)
		transport.BindHandler(func(ctx context.Context, transport api.WriterStream) error {
			return svc.Write(ctx, transport)
		})
	})
	Describe("Normal Operation", func() {
		It("Should write a segment to the storage", func() {
			client, err := transport.Stream(context.TODO(), "")
			Expect(err).ToNot(HaveOccurred())
			w, err := segment.NewWriter(client, keys.Strings()...)
			Expect(err).To(BeNil())
			Expect(w.Write([]api.Segment{{
				ChannelKey: keys[0].String(),
				Start:      telem.TimeStamp(0),
				Data:       []byte{1, 2, 3, 4, 5, 6, 7, 8},
			}})).To(BeTrue())
			Expect(w.Close()).To(BeNil())
		})
	})
	Describe("Invalid Arguments", func() {
		Context("Open", func() {
			Describe("No open keys provided", func() {
				It("Should return a validation error", func() {
					client, err := transport.Stream(context.TODO(), "")
					Expect(err).ToNot(HaveOccurred())
					w, err := segment.NewWriter(client)
					Expect(err).To(BeNil())
					Expect(w.Close()).To(Equal(errors.Validation(errors.Field{
						Field:   "open_keys",
						Message: "must contain at least one key",
					})))
				})
			})
			Describe("Invalid channel key provided", func() {
				It("Should return a validation error", func() {
					client, err := transport.Stream(context.TODO(), "")
					Expect(err).ToNot(HaveOccurred())
					w, err := segment.NewWriter(client, "invalid")
					Expect(err).To(BeNil())
					Expect(w.Close()).To(Equal(errors.Validation(errors.Field{
						Field:   "open_keys",
						Message: "[channel] - invalid key format",
					})))
				})
			})
			Describe("Lock already acquired", func() {
				It("Should return the correct error", func() {
					client1, err := transport.Stream(context.TODO(), "")
					Expect(err).ToNot(HaveOccurred())
					w1, err := segment.NewWriter(client1, keys.Strings()...)
					Expect(err).To(BeNil())
					client2, err := transport.Stream(context.TODO(), "")
					w2, err := segment.NewWriter(client2, keys.Strings()...)
					Expect(err).To(BeNil())
					expectedErr := errors.General(roacherrors.New("[cesium] - lock already held"))
					// expect one of the two writers to fail
					oneErr := false
					if err := w1.Close(); err != nil {
						oneErr = true
						Expect(err).To(Equal(expectedErr))
					}
					if err := w2.Close(); err != nil {
						oneErr = true
						Expect(err).To(Equal(expectedErr))
					}
					Expect(oneErr).To(BeTrue())
				})
			})
		})
		Context("Writing a segment", func() {
			Describe("Invalid channel key provided", func() {
				It("Should receive a validation error", func() {
					client, err := transport.Stream(context.TODO(), "")
					Expect(err).ToNot(HaveOccurred())
					w, err := segment.NewWriter(client, keys.Strings()...)
					Expect(err).To(BeNil())
					Expect(w.Write([]api.Segment{{
						ChannelKey: "invalid",
						Start:      telem.TimeStamp(0),
						Data:       []byte{1, 2, 3, 4, 5, 6, 7, 8},
					}})).To(BeTrue())
					Expect(w.Close()).To(Equal(errors.Validation(errors.Field{
						Field:   "channel_key",
						Message: "[channel] - invalid key format",
					})))
				})
			})
		})
	})
})
