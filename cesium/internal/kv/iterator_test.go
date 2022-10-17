package kv_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/kv"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("IteratorServer", func() {
	var (
		db *kv.DB
		ch channel.Channel
	)

	BeforeEach(func() {
		ch = channel.Channel{Key: 1, Density: 1}
		db = kv.Wrap(memkv.New())
		Expect(db.SetChannel(ch)).To(Succeed())
	})

	AfterEach(func() { Expect(db.Close()).To(Succeed()) })

	Context("Even Bounded TimeRange", func() {
		var iter *kv.mdIterator
		BeforeEach(func() {
			w := db.NewWriter()
			Expect(w.WriteMany([]kv.Segment{
				{
					ChannelKey: ch.Key,
					Start:      0,
					Size:       100,
				},
				{
					ChannelKey: ch.Key,
					Start:      100,
					Size:       100,
				},
			})).To(Succeed())
			Expect(w.Commit()).To(Succeed())
			var err error
			iter, err = db.NewIterator(ch.Key, position.Range{Start: 0, End: 200})
			Expect(err).ToNot(HaveOccurred())
		})
		AfterEach(func() { Expect(iter.Close()).To(Succeed()) })

		Describe("First", func() {
			It("Should return the correct segment", func() {
				Expect(iter.First()).To(BeTrue())
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.Error()).ToNot(HaveOccurred())
				Expect(iter.View()).To(Equal(position.Range{
					Start: 0,
					End:   100,
				}))
				Expect(iter.Value().Segments).To(HaveLen(1))
				Expect(iter.Value().BoundedRange()).To(Equal(iter.View()))
				Expect(iter.Close()).To(Succeed())
			})
		})

		Describe("Last", func() {
			It("Should return the correct segment", func() {
				Expect(iter.Last()).To(BeTrue())
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.Error()).ToNot(HaveOccurred())
				Expect(iter.View()).To(Equal(position.Range{
					Start: 100,
					End:   200,
				}))
				Expect(iter.Value().BoundedRange()).To(Equal(iter.View()))
				Expect(iter.Close()).To(Succeed())
			})
		})

		Describe("SeekFirst", func() {
			It("Should seek the correct view", func() {
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.Valid()).To(BeFalse())
				Expect(iter.Error()).ToNot(HaveOccurred())
				Expect(iter.View()).To(Equal(position.Range{Start: 0, End: 0}))
				Expect(iter.Value().Segments).To(HaveLen(1))
				Expect(iter.Value().BoundedRange()).To(Equal(iter.View()))
				Expect(iter.Close()).To(Succeed())
			})
		})

		Describe("SeekLast", func() {
			It("Should seek the correct view", func() {
				Expect(iter.SeekLast()).To(BeTrue())
				Expect(iter.Valid()).To(BeFalse())
				Expect(iter.Error()).ToNot(HaveOccurred())
				Expect(iter.View()).To(Equal(position.Position(200).SpanRange(0)))
			})
		})

		Describe("SeekLE", func() {

			Context("Within BoundedRange", func() {
				It("Should seek the correct view", func() {
					Expect(iter.Seek(50)).To(BeTrue())
					Expect(iter.Valid()).To(BeFalse())
					Expect(iter.Error()).ToNot(HaveOccurred())
				})
			})

			Context("Before BoundedRange", func() {
				It("Should return false", func() {
					Expect(iter.Seek(-5)).To(BeFalse())
					Expect(iter.Valid()).To(BeFalse())
					Expect(iter.Error()).ToNot(HaveOccurred())
				})
			})

			Context("After BoundedRange", func() {
				It("Should return true", func() {
					Expect(iter.Seek(500)).To(BeFalse())
					Expect(iter.Valid()).To(BeFalse())
					Expect(iter.Error()).ToNot(HaveOccurred())
				})
			})

			Describe("next", func() {
				It("Should return the segment that contains the view", func() {
					Expect(iter.Seek(50)).To(BeTrue())
					Expect(iter.Next()).To(BeTrue())
					Expect(iter.Value().Segments[0].Start).To(Equal(position.Position(0)))
				})
			})

		})

		Describe("next", func() {

			Context("At First", func() {
				It("Should return the correct segment", func() {
					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Next()).To(BeTrue())
					Expect(iter.Error()).ToNot(HaveOccurred())
					Expect(iter.View()).To(Equal(position.Range{
						Start: 0,
						End:   100,
					}))

				})
			})

			Context("At Last", func() {
				It("Should return false", func() {
					Expect(iter.SeekLast()).To(BeTrue())
					Expect(iter.Next()).To(BeFalse())
				})
			})

			Context("Approaching Last", func() {
				It("Should return false when it reaches last", func() {
					Expect(iter.First()).To(BeTrue())
					Expect(iter.Next()).To(BeTrue())
					Expect(iter.Next()).To(BeFalse())
				})
			})

		})

		Describe("Prev", func() {

			Context("At Last", func() {
				It("Should return the correct segment", func() {
					Expect(iter.SeekLast()).To(BeTrue())
					Expect(iter.Prev()).To(BeTrue())
					Expect(iter.Error()).ToNot(HaveOccurred())
					Expect(iter.View()).To(Equal(position.Range{
						Start: 100,
						End:   200,
					}))
				})
			})

			Context("At first", func() {
				It("Should return false", func() {
					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Prev()).To(BeFalse())
				})
			})

			Context("Approaching first", func() {
				It("Should return false when it reaches first", func() {
					Expect(iter.Last()).To(BeTrue())
					Expect(iter.Prev()).To(BeTrue())
					Expect(iter.Prev()).To(BeFalse())
				})
			})

		})

	})

	Context("Uneven Bounded TimeRange", func() {
		Context("First BoundedRange Starts After, Last BoundedRange Ends After", func() {
			var iter *kv.mdIterator
			BeforeEach(func() {
				w := db.NewWriter()
				Expect(w.WriteMany([]kv.Segment{
					{
						ChannelKey: ch.Key,
						Start:      10,
						Size:       100,
					},
					{
						ChannelKey: ch.Key,
						Start:      110,
						Size:       100,
					},
				})).To(Succeed())
				Expect(w.Commit()).To(Succeed())
				var err error
				iter, err = db.NewIterator(ch.Key, position.Range{
					Start: 5,
					End:   200,
				})
				Expect(err).ToNot(HaveOccurred())
			})
			AfterEach(func() { Expect(iter.Close()).To(Succeed()) })

			Describe("First", func() {
				It("Should return the correct segment", func() {
					Expect(iter.First()).To(BeTrue())
					Expect(iter.Valid()).To(BeTrue())
					Expect(iter.Error()).ToNot(HaveOccurred())
					Expect(iter.View()).To(Equal(position.Range{Start: 10, End: 110}))
					Expect(iter.Value().Segments).To(HaveLen(1))
					Expect(iter.Value().BoundedRange()).To(Equal(iter.View()))
					Expect(iter.Close()).To(Succeed())
				})
			})

			Describe("Last", func() {
				It("Should return the correct segment", func() {
					Expect(iter.Last()).To(BeTrue())
					Expect(iter.Valid()).To(BeTrue())
					Expect(iter.Error()).ToNot(HaveOccurred())
					Expect(iter.View()).To(Equal(position.Range{Start: 110, End: 200}))
					Expect(iter.Value().BoundedRange()).To(Equal(iter.View()))
				})
			})

			Describe("SeekTo", func() {
				It("Should seek to the correct view", func() {
					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Valid()).To(BeFalse())
					Expect(iter.Error()).ToNot(HaveOccurred())
					Expect(iter.View()).To(Equal(position.Position(10).SpanRange(0)))
				})

				It("Should return the correct next segment", func() {
					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Next()).To(BeTrue())
					Expect(iter.Value().Segments).To(HaveLen(1))
					Expect(iter.View()).To(Equal(position.Range{
						Start: 10,
						End:   110,
					}))
					Expect(iter.Value().BoundedRange()).To(Equal(iter.View()))
				})
			})

		})
	})

	Context("Invalid TimeRange", func() {
		var (
			err  error
			iter *kv.mdIterator
		)
		BeforeEach(func() {
			w := db.NewWriter()
			Expect(w.WriteMany([]kv.Segment{
				{
					ChannelKey: ch.Key,
					Start:      220,
					Size:       100,
				},
				{
					ChannelKey: ch.Key,
					Start:      320,
					Size:       100,
				},
			})).To(Succeed())
			Expect(w.Commit()).To(Succeed())
			iter, err = db.NewIterator(ch.Key, position.Range{Start: 5, End: 210})
		})

		Describe("First", func() {
			It("Should return false", func() {
				Expect(iter).To(BeNil())
				Expect(err).To(HaveOccurredAs(kv.RangeHasNoData))
			})
		})

	})

	Describe("NextSpan", func() {
		Context("Starting at the beginning of a segment", func() {
			Context("Reading an entire segment", func() {
				It("Should return the correct segment", func() {
					w := db.NewWriter()
					Expect(w.WriteMany([]kv.Segment{
						{
							ChannelKey: ch.Key,
							Start:      0,
							Size:       100,
						},
						{
							ChannelKey: ch.Key,
							Start:      100,
							Size:       100,
						},
					})).To(Succeed())
					Expect(w.Commit()).To(Succeed())
					iter, err := db.NewIterator(ch.Key, position.Range{Start: 0, End: 200})
					Expect(err).ToNot(HaveOccurred())
					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.NextSpan(100)).To(BeTrue())
					Expect(iter.Valid()).To(BeTrue())
					Expect(iter.Error()).ToNot(HaveOccurred())
					Expect(iter.View()).To(Equal(position.Range{
						Start: 0,
						End:   100,
					}))
					Expect(iter.Value().BoundedRange()).To(Equal(iter.View()))
					Expect(iter.Value().Segments).To(HaveLen(1))
					Expect(iter.Close()).To(Succeed())
				})
			})
			Context("Reading a partial segment", func() {

				It("Should return the partial segment correctly", func() {
					w := db.NewWriter()
					Expect(w.WriteMany([]kv.Segment{
						{
							ChannelKey: ch.Key,
							Start:      0,
							Size:       100,
						},
						{
							ChannelKey: ch.Key,
							Start:      100,
							Size:       100,
						},
					})).To(Succeed())
					Expect(w.Commit()).To(Succeed())
					iter, err := db.NewIterator(ch.Key, position.Range{Start: 0, End: 200})
					Expect(err).ToNot(HaveOccurred())
					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.NextSpan(50)).To(BeTrue())
					Expect(iter.Valid()).To(BeTrue())
					Expect(iter.Error()).ToNot(HaveOccurred())
					Expect(iter.View()).To(Equal(position.Range{
						Start: 0,
						End:   50,
					}))
					Expect(iter.Value().BoundedRange()).To(Equal(iter.View()))
					Expect(iter.Value().Segments).To(HaveLen(1))
					Expect(iter.Value().Segments[0].Start).To(Equal(position.Position(0)))
					Expect(iter.Prev()).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})
			})
		})

		Context("Reading multiple segments", func() {
			It("Should return the segments correctly", func() {
				w := db.NewWriter()
				Expect(w.WriteMany([]kv.Segment{
					{
						ChannelKey: ch.Key,
						Start:      0,
						Size:       100,
					},
					{
						ChannelKey: ch.Key,
						Start:      100,
						Size:       100,
					},
					{
						ChannelKey: ch.Key,
						Start:      200,
						Size:       100,
					},
				})).To(Succeed())
				Expect(w.Commit()).To(Succeed())
				iter, err := db.NewIterator(ch.Key, position.Range{Start: 0, End: 300})
				Expect(err).ToNot(HaveOccurred())
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.NextSpan(200)).To(BeTrue())
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.View()).To(Equal(position.Range{
					Start: 0,
					End:   200,
				}))
				Expect(iter.Value().BoundedRange()).To(Equal(iter.View()))
				Expect(iter.Value().Segments).To(HaveLen(2))
				Expect(iter.Close()).To(Succeed())
			})
		})

		Context("Crossing the global range boundary", func() {
			It("Should return a valid range of segments", func() {
				w := db.NewWriter()
				Expect(w.WriteMany([]kv.Segment{
					{
						ChannelKey: ch.Key,
						Start:      0,
						Size:       100,
					},
					{
						ChannelKey: ch.Key,
						Start:      100,
						Size:       100,
					},
					{
						ChannelKey: ch.Key,
						Start:      200,
						Size:       100,
					},
				})).To(Succeed())
				Expect(w.Commit()).To(Succeed())
				iter, err := db.NewIterator(ch.Key, position.Range{
					Start: 0,
					End:   300,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(iter.Seek(50)).To(BeTrue())
				Expect(iter.NextSpan(300)).To(BeTrue())
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.View()).To(Equal(position.Range{
					Start: 50,
					End:   300,
				}))
				Expect(iter.Value().BoundedRange()).To(Equal(iter.View()))
				Expect(iter.Value().Segments).To(HaveLen(3))

				By("Returning false on the next call to next()")
				Expect(iter.Next()).To(BeFalse())
				Expect(iter.Valid()).To(BeFalse())

				Expect(iter.NextSpan(30)).To(BeFalse())
				Expect(iter.Valid()).To(BeFalse())

				Expect(iter.Close()).To(Succeed())
			})
		})

		Context("Crossing the global range boundary twice", func() {
			It("Should return false", func() {
				w := db.NewWriter()
				Expect(w.WriteMany([]kv.Segment{
					{
						ChannelKey: ch.Key,
						Start:      0,
						Size:       100,
					},
					{
						ChannelKey: ch.Key,
						Start:      100,
						Size:       100,
					},
					{
						ChannelKey: ch.Key,
						Start:      200,
						Size:       100,
					},
				})).To(Succeed())
				Expect(w.Commit()).To(Succeed())
				iter, err := db.NewIterator(ch.Key, position.Range{Start: 0, End: 300})
				Expect(err).ToNot(HaveOccurred())
				Expect(iter.Seek(50)).To(BeTrue())
				Expect(iter.NextSpan(300)).To(BeTrue())
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.View()).To(Equal(position.Range{
					Start: 50,
					End:   300,
				}))
				Expect(iter.Value().BoundedRange()).To(Equal(iter.View()))
				Expect(iter.Value().Segments).To(HaveLen(3))

				Expect(iter.NextSpan(30)).To(BeFalse())
				Expect(iter.Valid()).To(BeFalse())

				Expect(iter.Close()).To(Succeed())
			})
		})

	})

	Context("Sequences", func() {

		Context("Contiguous, Even BoundedRange", func() {
			It("Should move the iterator view correctly", func() {

				w := db.NewWriter()
				Expect(w.WriteMany([]kv.Segment{
					{
						ChannelKey: ch.Key,
						Start:      0,
						Size:       100,
					},
					{
						ChannelKey: ch.Key,
						Start:      100,
						Size:       100,
					},
					{
						ChannelKey: ch.Key,
						Start:      200,
						Size:       100,
					},
				})).To(Succeed())
				Expect(w.Commit()).To(Succeed())
				iter, err := db.NewIterator(ch.Key, position.Range{Start: 0, End: 300})
				Expect(err).ToNot(HaveOccurred())

				By("Moving to the first value")
				Expect(iter.First()).To(BeTrue())

				By("Moving over the next span")
				Expect(iter.NextSpan(25)).To(BeTrue())
				Expect(iter.View()).To(Equal(position.Range{
					Start: 100,
					End:   125,
				}))
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.Value().Segments).To(HaveLen(1))
				Expect(iter.Value().Segments[0].Start).To(Equal(position.Position(100)))
				Expect(iter.Value().BoundedRange()).To(Equal(iter.View()))

				By("Reversing over a span")
				Expect(iter.PrevSpan(25)).To(BeTrue())
				Expect(iter.View()).To(Equal(position.Range{
					Start: 75,
					End:   100,
				}))
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.Value().Segments).To(HaveLen(1))
				Expect(iter.Value().BoundedRange()).To(Equal(iter.View()))

				By("Reversing over a span again")
				Expect(iter.PrevSpan(25)).To(BeTrue())
				Expect(iter.View()).To(Equal(position.Range{
					Start: 50,
					End:   75,
				}))
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.Value().Segments).To(HaveLen(1))
				Expect(iter.Value().Segments[0].Start).To(Equal(position.Position(0)))
				Expect(iter.Value().BoundedRange()).To(Equal(iter.View()))

				By("Reversing over the global range boundary")
				Expect(iter.PrevSpan(100)).To(BeTrue())
				Expect(iter.View()).To(Equal(position.Range{
					Start: 0,
					End:   50,
				}))
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.Value().Segments).To(HaveLen(1))

				By("Reversing over the global range boundary again")
				Expect(iter.PrevSpan(100)).To(BeFalse())
				Expect(iter.Valid()).To(BeFalse())

				By("Moving forward over the next span")
				Expect(iter.NextSpan(100)).To(BeTrue())
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.View()).To(Equal(position.Range{Start: 50, End: 150}))
				Expect(iter.Value().Segments).To(HaveLen(2))
				Expect(iter.Valid()).To(BeTrue())

				By("Seeking over a range")
				nextRange := position.Range{Start: 0, End: 50}
				Expect(iter.SetRange(nextRange)).To(BeTrue())
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.View()).To(Equal(nextRange))
				Expect(iter.Value().BoundedRange()).To(Equal(nextRange))
				Expect(iter.Close()).To(Succeed())
			})

		})

		Context("Non-Contiguous, Uneven BoundedRange", func() {
			It("Should move the iterator view correctly", func() {
				w := db.NewWriter()
				Expect(w.WriteMany([]kv.Segment{
					{
						ChannelKey: ch.Key,
						Start:      10,
						Size:       20,
					},
					{
						ChannelKey: ch.Key,
						Start:      50,
						Size:       100,
					},
					{
						ChannelKey: ch.Key,
						Start:      150,
						Size:       100,
					},
				}))
				Expect(w.Commit()).To(Succeed())

				iter, err := db.NewIterator(ch.Key, position.Range{Start: 0, End: 300})
				Expect(err).ToNot(HaveOccurred())

				By("Moving to the last value")
				Expect(iter.Last()).To(BeTrue())
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.View()).To(Equal(position.Range{
					Start: 150,
					End:   250,
				}))

				By("Moving to an empty section of data")
				Expect(iter.SetRange(position.Range{
					Start: 30,
					End:   50,
				}))
				Expect(iter.Valid()).To(BeFalse())
				Expect(iter.Value().BoundedRange().Span().IsZero()).To(BeTrue())

				By("Moving to the next span")
				Expect(iter.NextSpan(50)).To(BeTrue())
				Expect(iter.View()).To(Equal(position.Range{
					Start: 50,
					End:   100,
				}))
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.Value().BoundedRange()).To(Equal(iter.View()))

				Expect(iter.Close()).To(Succeed())

			})
		})

	})
})
