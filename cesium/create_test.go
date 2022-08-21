package cesium_test

import (
	"github.com/arya-analytics/cesium"
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("Create", func() {
	var (
		db cesium.DB
	)
	BeforeEach(func() {
		var err error
		log := zap.NewNop()
		db, err = cesium.Open("", cesium.MemBacked(), cesium.WithLogger(log))
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(db.Close()).To(Succeed())
	})

	Describe("Basic Functionality", func() {

		Context("Single channel", func() {

			Context("Single segment", func() {

				It("Should write the segment correctly", func() {

					By("Creating a new channel")
					key, err := db.CreateChannel(cesium.Channel{
						DataRate: 1 * cesium.Hz,
						DataType: cesium.Float64,
					})
					Expect(err).ToNot(HaveOccurred())

					By("Initializing a request")
					segments := []cesium.Segment{
						{
							ChannelKey: key,
							Start:      cesium.Now(),
							Data:       []byte{1, 2, 3, 4, 5, 6, 7, 8},
						},
					}

					By("Opening the create query")
					Expect(db.Sync(ctx, db.NewCreate().WhereChannels(key),
						&segments)).To(Succeed())
					Expect(err).ToNot(HaveOccurred())

					var resSeg []cesium.Segment
					err = db.Sync(ctx, db.NewRetrieve().
						WhereChannels(key).
						WhereTimeRange(cesium.TimeRangeMax),
						&resSeg,
					)
					Expect(err).ToNot(HaveOccurred())
					Expect(resSeg).To(HaveLen(1))
					Expect(resSeg[0].Start).To(Equal(segments[0].Start))
				})
			})

			Context("Multi segment", func() {

				It("Should write the segments correctly", func() {
					By("Creating a new channel")
					key, err := db.CreateChannel(cesium.Channel{
						DataRate: 1 * cesium.Hz,
						DataType: cesium.Float64,
					})
					Expect(err).ToNot(HaveOccurred())

					By("Initializing a request")
					segments := []cesium.Segment{
						{
							ChannelKey: key,
							Start:      cesium.Now(),
							Data:       []byte{1, 2, 3, 4, 5, 6, 7, 8},
						},
						{
							ChannelKey: key,
							Start:      cesium.Now().Add(1 * cesium.Second),
							Data:       []byte{1, 2, 3, 4, 5, 6, 7, 8},
						},
					}

					By("Opening the create query")
					Expect(db.Sync(ctx, db.NewCreate().WhereChannels(key), &segments)).To(Succeed())

					By("Retrieving the segments afterwards")
					var resSeg []cesium.Segment
					err = db.Sync(ctx, db.NewRetrieve().WhereChannels(key).WhereTimeRange(cesium.TimeRangeMax), &resSeg)
					Expect(err).ToNot(HaveOccurred())
					Expect(resSeg).To(HaveLen(2))
					ts := []cesium.TimeStamp{segments[0].Start, segments[1].Start}
					Expect(resSeg[0].Start).To(BeElementOf(ts))
					Expect(resSeg[1].Start).To(BeElementOf(ts))
				})
			})

			Context("Multi Request", func() {

				It("Should write the segments correctly", func() {
					By("Creating a new channel")
					key, err := db.CreateChannel(cesium.Channel{
						DataRate: 1 * cesium.Hz,
						DataType: cesium.Float64,
					})
					Expect(err).ToNot(HaveOccurred())

					By("Initializing a request")
					cReqOne := cesium.CreateRequest{
						Segments: []cesium.Segment{
							{
								ChannelKey: key,
								Start:      cesium.Now(),
								Data:       []byte{1, 2, 3, 4, 5, 6, 7, 8},
							},
						},
					}
					cReqTwo := cesium.CreateRequest{
						Segments: []cesium.Segment{
							{
								ChannelKey: key,
								Start:      cesium.Now().Add(1 * cesium.Second),
								Data:       []byte{1, 2, 3, 4, 5, 6, 7, 8},
							},
						},
					}

					By("Opening the create query")
					req, res, err := db.NewCreate().WhereChannels(key).Stream(ctx)

					By("Writing the segments")
					req <- cReqOne
					req <- cReqTwo

					By("Closing the request pipe")
					close(req)

					By("Not returning any errors")
					for resV := range res {
						Expect(resV.Error).ToNot(HaveOccurred())
					}

					By("Retrieving the segments afterwards")
					var resSeg []cesium.Segment
					err = db.Sync(ctx, db.NewRetrieve().WhereChannels(key).WhereTimeRange(cesium.TimeRangeMax), &resSeg)
					Expect(err).ToNot(HaveOccurred())
					Expect(resSeg).To(HaveLen(2))
					ts := []cesium.TimeStamp{
						cReqOne.Segments[0].Start,
						cReqTwo.Segments[0].Start}
					Expect(resSeg[0].Start).To(BeElementOf(ts))
					Expect(resSeg[1].Start).To(BeElementOf(ts))
				})

			})

		})

	})

	Describe("Write TryLock Violation", func() {
		It("Should return an error when another query has a write lock on the channel", func() {
			By("Creating a new channel")
			key, err := db.CreateChannel(cesium.Channel{
				DataRate: 1 * cesium.Hz,
				DataType: cesium.Float64,
			})
			Expect(err).ToNot(HaveOccurred())

			By("Opening the first query")
			_, _, err = db.NewCreate().WhereChannels(key).Stream(ctx)
			Expect(err).ToNot(HaveOccurred())

			By("Failing to open the second query")
			_, _, err = db.NewCreate().WhereChannels(key).Stream(ctx)
			Expect(errors.Is(err, errors.New("[cesium] - lock already held"))).To(BeTrue())
		})
	})

})
