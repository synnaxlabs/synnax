package kv_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/kv"
	"github.com/synnaxlabs/cesium/internal/segment"
	kvx "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/telem"
	"time"
)

var _ = Describe("IteratorServer", func() {
	var (
		kve      kvx.DB
		ch       channel.Channel
		chTwo    channel.Channel
		chThree  channel.Channel
		headerKV *kv.Header
	)

	BeforeEach(func() {
		ch = channel.Channel{Key: 1, Rate: 1, Density: 1}
		chTwo = channel.Channel{Key: 2, Rate: 1, Density: 1}
		chThree = channel.Channel{Key: 3, Rate: 1, Density: 1}
		kve = memkv.New()
		headerKV = kv.NewHeader(kve)
		chKV := kv.NewChannelService(kve)
		Expect(chKV.Set(ch)).To(Succeed())
		Expect(chKV.Set(chTwo)).To(Succeed())
		Expect(chKV.Set(chThree)).To(Succeed())
	})

	AfterEach(func() { Expect(kve.Close()).To(Succeed()) })

	Context("Even Bounded TimeRange", func() {
		var iter kv.Iterator
		BeforeEach(func() {
			Expect(headerKV.Set(segment.Header{
				ChannelKey: ch.Key,
				Start:      0,
				Size:       100,
			})).To(Succeed())
			Expect(headerKV.Set(segment.Header{
				ChannelKey: ch.Key,
				Start:      telem.TimeStamp(100 * telem.Second),
				Size:       100,
			})).To(Succeed())
			var err error
			iter, err = kv.NewIterator(kve, telem.TimeRange{
				Start: 0,
				End:   telem.TimeStamp(200 * time.Second),
			}, ch.Key)
			Expect(err).ToNot(HaveOccurred())
		})
		AfterEach(func() { Expect(iter.Close()).To(Succeed()) })

		Describe("First", func() {
			It("Should return the correct segment", func() {
				Expect(iter.First()).To(BeTrue())
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.Error()).ToNot(HaveOccurred())
				Expect(iter.View()).To(Equal(telem.TimeRange{
					Start: 0,
					End:   telem.TimeStamp(100 * telem.Second),
				}))
				Expect(iter.Range().Headers).To(HaveLen(1))
				Expect(iter.Range().Range()).To(Equal(iter.View()))
				Expect(iter.Close()).To(Succeed())
			})
		})

		Describe("Last", func() {
			It("Should return the correct segment", func() {
				Expect(iter.Last()).To(BeTrue())
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.Error()).ToNot(HaveOccurred())
				Expect(iter.View()).To(Equal(telem.TimeRange{
					Start: telem.TimeStamp(100 * telem.Second),
					End:   telem.TimeStamp(200 * time.Second),
				}))
				Expect(iter.Range().Range()).To(Equal(iter.View()))
				Expect(iter.Close()).To(Succeed())
			})
		})

		Describe("SeekFirst", func() {
			It("Should seek the correct view", func() {
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.Valid()).To(BeFalse())
				Expect(iter.Error()).ToNot(HaveOccurred())
				Expect(iter.View()).To(Equal(telem.TimeRange{Start: 0, End: 0}))
				Expect(iter.Range().Headers).To(HaveLen(1))
				Expect(iter.Range().Range()).To(Equal(iter.View()))
				Expect(iter.Close()).To(Succeed())
			})
		})

		Describe("SeekLast", func() {
			It("Should seek the correct view", func() {
				Expect(iter.SeekLast()).To(BeTrue())
				Expect(iter.Valid()).To(BeFalse())
				Expect(iter.Error()).ToNot(HaveOccurred())
				Expect(iter.View()).To(Equal(telem.TimeStamp(200 * time.Second).SpanRange(0)))
			})
		})

		Describe("Seek", func() {

			Context("Within BoundedRange", func() {
				It("Should seek the correct view", func() {
					Expect(iter.Seek(telem.TimeStamp(50 * time.Second))).To(BeTrue())
					Expect(iter.Valid()).To(BeFalse())
					Expect(iter.Error()).ToNot(HaveOccurred())
				})
			})

			Context("Before BoundedRange", func() {
				It("Should return false", func() {
					Expect(iter.Seek(telem.TimeStamp(-5 * time.Second))).To(BeFalse())
					Expect(iter.Valid()).To(BeFalse())
					Expect(iter.Error()).ToNot(HaveOccurred())
				})
			})

			Context("After BoundedRange", func() {
				It("Should return true", func() {
					Expect(iter.Seek(telem.TimeStamp(500 * time.Second))).To(BeFalse())
					Expect(iter.Valid()).To(BeFalse())
					Expect(iter.Error()).ToNot(HaveOccurred())
				})
			})

			Describe("next", func() {
				It("Should return the segment that contains the view", func() {
					Expect(iter.Seek(telem.TimeStamp(50 * time.Second))).To(BeTrue())
					Expect(iter.Next()).To(BeTrue())
					Expect(iter.Range().Headers[0].Start).To(Equal(telem.TimeStamp(0 * time.Second)))
				})
			})

		})

		Describe("next", func() {

			Context("At First", func() {
				It("Should return the correct segment", func() {
					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Next()).To(BeTrue())
					Expect(iter.Error()).ToNot(HaveOccurred())
					Expect(iter.View()).To(Equal(telem.TimeRange{
						Start: 0,
						End:   telem.TimeStamp(100 * time.Second),
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
					Expect(iter.View()).To(Equal(telem.TimeRange{
						Start: telem.TimeStamp(100 * time.Second),
						End:   telem.TimeStamp(200 * time.Second),
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

			var iter kv.Iterator
			BeforeEach(func() {
				Expect(headerKV.SetMultiple([]segment.Header{
					{
						ChannelKey: ch.Key,
						Start:      telem.TimeStamp(10 * time.Second),
						Size:       100,
					},
					{
						ChannelKey: ch.Key,
						Start:      telem.TimeStamp(110 * time.Second),
						Size:       100,
					},
				})).To(Succeed())
				var err error
				iter, err = kv.NewIterator(kve, telem.TimeRange{
					Start: telem.TimeStamp(5 * time.Second),
					End:   telem.TimeStamp(200 * time.Second),
				}, ch.Key)
				Expect(err).ToNot(HaveOccurred())
			})
			AfterEach(func() { Expect(iter.Close()).To(Succeed()) })

			Describe("First", func() {
				It("Should return the correct segment", func() {
					Expect(iter.First()).To(BeTrue())
					Expect(iter.Valid()).To(BeTrue())
					Expect(iter.Error()).ToNot(HaveOccurred())
					Expect(iter.View()).To(Equal(telem.TimeRange{
						Start: telem.TimeStamp(10 * time.Second),
						End:   telem.TimeStamp(110 * time.Second),
					}))
					Expect(iter.Range().Headers).To(HaveLen(1))
					Expect(iter.Range().Range()).To(Equal(iter.View()))
					Expect(iter.Close()).To(Succeed())
				})
			})

			Describe("Last", func() {
				It("Should return the correct segment", func() {
					Expect(iter.Last()).To(BeTrue())
					Expect(iter.Valid()).To(BeTrue())
					Expect(iter.Error()).ToNot(HaveOccurred())
					Expect(iter.View()).To(Equal(telem.TimeRange{
						Start: telem.TimeStamp(110 * time.Second),
						End:   telem.TimeStamp(200 * time.Second),
					}))
					Expect(iter.Range().Range()).To(Equal(iter.View()))
				})
			})

			Describe("SeekTo", func() {
				It("Should seek to the correct view", func() {
					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Valid()).To(BeFalse())
					Expect(iter.Error()).ToNot(HaveOccurred())
					Expect(iter.View()).To(Equal(telem.TimeStamp(10 * time.Second).SpanRange(0)))
				})

				It("Should return the correct next segment", func() {
					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Next()).To(BeTrue())
					Expect(iter.Range().Headers).To(HaveLen(1))
					Expect(iter.View()).To(Equal(telem.TimeRange{
						Start: telem.TimeStamp(10 * time.Second),
						End:   telem.TimeStamp(110 * time.Second),
					}))
					Expect(iter.Range().Range()).To(Equal(iter.View()))
				})
			})

		})
	})

	Context("Invalid Bounded TimeRange", func() {
		var (
			err  error
			iter kv.Iterator
		)
		BeforeEach(func() {
			Expect(headerKV.SetMultiple([]segment.Header{
				{
					ChannelKey: ch.Key,
					Start:      telem.TimeStamp(220 * time.Second),
					Size:       100,
				},
				{
					ChannelKey: ch.Key,
					Start:      telem.TimeStamp(320 * time.Second),
					Size:       100,
				},
			})).To(Succeed())
			iter, err = kv.NewIterator(kve, telem.TimeRange{
				Start: telem.TimeStamp(5 * time.Second),
				End:   telem.TimeStamp(210 * time.Second),
			}, ch.Key)
		})

		Describe("First", func() {
			It("Should return false", func() {
				Expect(iter).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("[cesium.kv] - range has no data"))
			})
		})

	})

	Context("Current Timestamp", func() {
		var (
			iter kv.Iterator
			ts   telem.TimeStamp
		)
		BeforeEach(func() {
			ts = telem.Now()
			Expect(headerKV.SetMultiple([]segment.Header{
				{
					ChannelKey: ch.Key,
					Start:      ts,
					Size:       1,
				},
				{
					ChannelKey: ch.Key,
					Start:      ts.Add(1 * telem.Second),
					Size:       1,
				},
			})).To(Succeed())
			var err error
			iter, err = kv.NewIterator(kve, telem.TimeRange{
				Start: telem.Now().Sub(10 * telem.Second),
				End:   telem.Now().Add(10 * telem.Second),
			}, ch.Key)
			Expect(err).ToNot(HaveOccurred())
		})
		AfterEach(func() { Expect(iter.Close()).To(Succeed()) })
		It("Should open the iterator without error", func() {
			Expect(iter.Error()).To(BeNil())
		})
		It("Should return the correct segments", func() {
			Expect(iter.First()).To(BeTrue())
			Expect(iter.Valid()).To(BeTrue())
			Expect(iter.Error()).ToNot(HaveOccurred())
			Expect(iter.Range().Headers).To(HaveLen(1))
			Expect(iter.Range().Range()).To(Equal(telem.TimeRange{
				Start: ts,
				End:   ts.Add(1 * telem.Second),
			}))
			Expect(iter.Next()).To(BeTrue())
			Expect(iter.Valid()).To(BeTrue())
			Expect(iter.View()).To(Equal(telem.TimeRange{
				Start: ts.Add(1 * telem.Second),
				End:   ts.Add(2 * telem.Second),
			}))
		})
	})

	Describe("NextSpan", func() {
		Context("Starting at the beginning of a segment", func() {
			Context("Reading an entire segment", func() {
				It("Should return the correct segment", func() {
					Expect(headerKV.SetMultiple([]segment.Header{
						{
							ChannelKey: ch.Key,
							Start:      0,
							Size:       100,
						},
						{
							ChannelKey: ch.Key,
							Start:      telem.TimeStamp(100 * telem.Second),
							Size:       100,
						},
					})).To(Succeed())
					iter, err := kv.NewIterator(kve, telem.TimeRange{
						Start: 0,
						End:   telem.TimeStamp(200 * time.Second),
					}, ch.Key)
					Expect(err).ToNot(HaveOccurred())
					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.NextSpan(100 * telem.Second)).To(BeTrue())
					Expect(iter.Valid()).To(BeTrue())
					Expect(iter.Error()).ToNot(HaveOccurred())
					Expect(iter.View()).To(Equal(telem.TimeRange{
						Start: 0,
						End:   telem.TimeStamp(100 * telem.Second),
					}))
					Expect(iter.Range().Range()).To(Equal(iter.View()))
					Expect(iter.Range().Headers).To(HaveLen(1))
					Expect(iter.Close()).To(Succeed())
				})
			})
			Context("Reading a partial segment", func() {

				It("Should return the partial segment correctly", func() {
					Expect(headerKV.SetMultiple([]segment.Header{
						{
							ChannelKey: ch.Key,
							Start:      0,
							Size:       100,
						},
						{
							ChannelKey: ch.Key,
							Start:      telem.TimeStamp(100 * telem.Second),
							Size:       100,
						},
					})).To(Succeed())
					iter, err := kv.NewIterator(kve, telem.TimeRange{
						Start: 0,
						End:   telem.TimeStamp(200 * time.Second),
					}, ch.Key)
					Expect(err).ToNot(HaveOccurred())
					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.NextSpan(50 * telem.Second)).To(BeTrue())
					Expect(iter.Valid()).To(BeTrue())
					Expect(iter.Error()).ToNot(HaveOccurred())
					Expect(iter.View()).To(Equal(telem.TimeRange{
						Start: 0,
						End:   telem.TimeStamp(50 * telem.Second),
					}))
					Expect(iter.Range().Range()).To(Equal(iter.View()))
					Expect(iter.Range().Headers).To(HaveLen(1))
					Expect(iter.Range().Headers[0].Start).To(Equal(telem.TimeStamp(0)))
					Expect(iter.Prev()).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})
			})
		})

		Context("Reading multiple segments", func() {
			It("Should return the segments correctly", func() {
				Expect(headerKV.SetMultiple([]segment.Header{
					{
						ChannelKey: ch.Key,
						Start:      0,
						Size:       100,
					},
					{
						ChannelKey: ch.Key,
						Start:      telem.TimeStamp(100 * telem.Second),
						Size:       100,
					},
					{
						ChannelKey: ch.Key,
						Start:      telem.TimeStamp(200 * time.Second),
						Size:       100,
					},
				})).To(Succeed())
				iter, err := kv.NewIterator(kve, telem.TimeRange{
					Start: 0,
					End:   telem.TimeStamp(300 * time.Second),
				}, ch.Key)
				Expect(err).ToNot(HaveOccurred())
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.NextSpan(200 * telem.Second)).To(BeTrue())
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.View()).To(Equal(telem.TimeRange{
					Start: 0,
					End:   telem.TimeStamp(200 * time.Second),
				}))
				Expect(iter.Range().Range()).To(Equal(iter.View()))
				Expect(iter.Range().Headers).To(HaveLen(2))
				Expect(iter.Close()).To(Succeed())
			})
		})

		Context("Crossing the global range boundary", func() {
			It("Should return a valid range of segments", func() {

				Expect(headerKV.SetMultiple([]segment.Header{
					{
						ChannelKey: ch.Key,
						Start:      0,
						Size:       100,
					},
					{
						ChannelKey: ch.Key,
						Start:      telem.TimeStamp(100 * telem.Second),
						Size:       100,
					},
					{
						ChannelKey: ch.Key,
						Start:      telem.TimeStamp(200 * time.Second),
						Size:       100,
					},
				})).To(Succeed())
				iter, err := kv.NewIterator(kve, telem.TimeRange{
					Start: 0,
					End:   telem.TimeStamp(300 * time.Second),
				}, ch.Key)
				Expect(err).ToNot(HaveOccurred())
				Expect(iter.Seek(telem.TimeStamp(50 * time.Second))).To(BeTrue())
				Expect(iter.NextSpan(300 * telem.Second)).To(BeTrue())
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.View()).To(Equal(telem.TimeRange{
					Start: telem.TimeStamp(50 * telem.Second),
					End:   telem.TimeStamp(300 * telem.Second),
				}))
				Expect(iter.Range().Range()).To(Equal(iter.View()))
				Expect(iter.Range().Headers).To(HaveLen(3))

				By("Returning false on the next call to next()")
				Expect(iter.Next()).To(BeFalse())
				Expect(iter.Valid()).To(BeFalse())

				Expect(iter.NextSpan(30 * telem.Second)).To(BeFalse())
				Expect(iter.Valid()).To(BeFalse())

				Expect(iter.Close()).To(Succeed())
			})
		})

		Context("Crossing the global range boundary twice", func() {
			It("Should return false", func() {
				Expect(headerKV.SetMultiple([]segment.Header{
					{
						ChannelKey: ch.Key,
						Start:      0,
						Size:       100,
					},
					{
						ChannelKey: ch.Key,
						Start:      telem.TimeStamp(100 * telem.Second),
						Size:       100,
					},
					{
						ChannelKey: ch.Key,
						Start:      telem.TimeStamp(200 * time.Second),
						Size:       100,
					},
				})).To(Succeed())
				iter, err := kv.NewIterator(kve, telem.TimeRange{
					Start: 0,
					End:   telem.TimeStamp(300 * time.Second),
				}, ch.Key)
				Expect(err).ToNot(HaveOccurred())
				Expect(iter.Seek(telem.TimeStamp(50 * time.Second))).To(BeTrue())
				Expect(iter.NextSpan(300 * telem.Second)).To(BeTrue())
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.View()).To(Equal(telem.TimeRange{
					Start: telem.TimeStamp(50 * telem.Second),
					End:   telem.TimeStamp(300 * telem.Second),
				}))
				Expect(iter.Range().Range()).To(Equal(iter.View()))
				Expect(iter.Range().Headers).To(HaveLen(3))

				Expect(iter.NextSpan(30 * telem.Second)).To(BeFalse())
				Expect(iter.Valid()).To(BeFalse())

				Expect(iter.Close()).To(Succeed())
			})
		})

	})

	Context("Sequences", func() {

		Context("Contiguous, Even BoundedRange", func() {
			It("Should move the iterator view correctly", func() {
				Expect(headerKV.SetMultiple([]segment.Header{
					{
						ChannelKey: ch.Key,
						Start:      0,
						Size:       100,
					},
					{
						ChannelKey: ch.Key,
						Start:      telem.TimeStamp(100 * telem.Second),
						Size:       100,
					},
					{
						ChannelKey: ch.Key,
						Start:      telem.TimeStamp(200 * time.Second),
						Size:       100,
					},
				})).To(Succeed())
				iter, err := kv.NewIterator(kve, telem.TimeRange{
					Start: 0,
					End:   telem.TimeStamp(300 * time.Second),
				}, ch.Key)
				Expect(err).ToNot(HaveOccurred())

				By("Moving to the first value")
				Expect(iter.First()).To(BeTrue())

				By("Moving over the next span")
				Expect(iter.NextSpan(25 * telem.Second)).To(BeTrue())
				Expect(iter.View()).To(Equal(telem.TimeRange{
					Start: telem.TimeStamp(100 * telem.Second),
					End:   telem.TimeStamp(125 * telem.Second),
				}))
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.Range().Headers).To(HaveLen(1))
				Expect(iter.Range().Headers[0].Start).To(Equal(telem.TimeStamp(100 * telem.Second)))
				Expect(iter.Range().Range()).To(Equal(iter.View()))

				By("Reversing over a span")
				Expect(iter.PrevSpan(25 * telem.Second)).To(BeTrue())
				Expect(iter.View()).To(Equal(telem.TimeRange{
					Start: telem.TimeStamp(75 * telem.Second),
					End:   telem.TimeStamp(100 * telem.Second),
				}))
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.Range().Headers).To(HaveLen(1))
				Expect(iter.Range().Range()).To(Equal(iter.View()))

				By("Reversing over a span again")
				Expect(iter.PrevSpan(25 * telem.Second)).To(BeTrue())
				Expect(iter.View()).To(Equal(telem.TimeRange{
					Start: telem.TimeStamp(50 * telem.Second),
					End:   telem.TimeStamp(75 * telem.Second),
				}))
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.Range().Headers).To(HaveLen(1))
				Expect(iter.Range().Headers[0].Start).To(Equal(telem.TimeStamp(0)))
				Expect(iter.Range().Range()).To(Equal(iter.View()))

				By("Reversing over the global range boundary")
				Expect(iter.PrevSpan(100 * telem.Second)).To(BeTrue())
				Expect(iter.View()).To(Equal(telem.TimeRange{
					Start: telem.TimeStamp(0 * telem.Second),
					End:   telem.TimeStamp(50 * telem.Second),
				}))
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.Range().Headers).To(HaveLen(1))

				By("Reversing over the global range boundary again")
				Expect(iter.PrevSpan(100 * telem.Second)).To(BeFalse())
				Expect(iter.Valid()).To(BeFalse())

				By("Moving forward over the next span")
				Expect(iter.NextSpan(100 * telem.Second)).To(BeTrue())
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.View()).To(Equal(telem.TimeRange{
					Start: telem.TimeStamp(50 * telem.Second),
					End:   telem.TimeStamp(150 * telem.Second),
				}))
				Expect(iter.Range().Headers).To(HaveLen(2))
				Expect(iter.Valid()).To(BeTrue())

				By("Seeking over a range")
				nextRange := telem.TimeRange{
					Start: 0,
					End:   telem.TimeStamp(50 * telem.Second),
				}
				Expect(iter.SetRange(nextRange)).To(BeTrue())
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.View()).To(Equal(nextRange))
				Expect(iter.Range().Range()).To(Equal(nextRange))
				Expect(iter.Close()).To(Succeed())
			})

		})

		Context("Non-Contiguous, Uneven BoundedRange", func() {
			It("Should move the iterator view correctly", func() {
				Expect(headerKV.SetMultiple([]segment.Header{
					{
						ChannelKey: ch.Key,
						Start:      telem.TimeStamp(10 * telem.Second),
						Size:       20,
					},
					{
						ChannelKey: ch.Key,
						Start:      telem.TimeStamp(50 * telem.Second),
						Size:       100,
					},
					{
						ChannelKey: ch.Key,
						Start:      telem.TimeStamp(150 * telem.Second),
						Size:       100,
					},
				}))

				iter, err := kv.NewIterator(kve, telem.TimeRange{
					Start: 0,
					End:   telem.TimeStamp(300 * telem.Second),
				}, ch.Key)
				Expect(err).ToNot(HaveOccurred())

				By("Moving to the last value")
				Expect(iter.Last()).To(BeTrue())
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.View()).To(Equal(telem.TimeRange{
					Start: telem.TimeStamp(150 * telem.Second),
					End:   telem.TimeStamp(250 * telem.Second),
				}))

				By("Moving to an empty section of data")
				Expect(iter.SetRange(telem.TimeRange{
					Start: telem.TimeStamp(30 * telem.Second),
					End:   telem.TimeStamp(50 * telem.Second),
				}))
				Expect(iter.Valid()).To(BeFalse())
				Expect(iter.Range().Range().Span().IsZero()).To(BeTrue())

				By("Moving to the next span")
				Expect(iter.NextSpan(50 * telem.Second)).To(BeTrue())
				Expect(iter.View()).To(Equal(telem.TimeRange{
					Start: telem.TimeStamp(50 * telem.Second),
					End:   telem.TimeStamp(100 * telem.Second),
				}))
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.Range().Range()).To(Equal(iter.View()))

				Expect(iter.Close()).To(Succeed())

			})
		})

	})

	Context("Compound IteratorServer", func() {

		Context("Sequences", func() {
			It("Should move the iterator view correctly", func() {
				keys := []channel.Key{chThree.Key, chTwo.Key, ch.Key}
				for _, k := range keys {
					k := k
					Expect(headerKV.SetMultiple([]segment.Header{
						{
							ChannelKey: k,
							Start:      0,
							Size:       100,
						},
						{
							ChannelKey: k,
							Start:      telem.TimeStamp(100 * telem.Second),
							Size:       100,
						},
						{
							ChannelKey: k,
							Start:      telem.TimeStamp(200 * time.Second),
							Size:       100,
						},
						{
							ChannelKey: k,
							Start:      telem.TimeStamp(300 * time.Second),
							Size:       100,
						},
					})).To(Succeed())
				}

				iter, err := kv.NewIterator(kve, telem.TimeRange{
					Start: 0,
					End:   telem.TimeStamp(400 * telem.Second),
				}, chThree.Key, chTwo.Key, ch.Key)
				Expect(err).ToNot(HaveOccurred())

				By("Moving to the first value")
				Expect(iter.First()).To(BeTrue())

				By("Moving over the next span")
				Expect(iter.NextSpan(25 * telem.Second)).To(BeTrue())
				Expect(iter.View()).To(Equal(telem.TimeRange{
					Start: telem.TimeStamp(100 * telem.Second),
					End:   telem.TimeStamp(125 * telem.Second),
				}))
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.Ranges()).To(HaveLen(3))
				Expect(iter.Ranges()[0].Headers).To(HaveLen(1))
				Expect(iter.Ranges()[1].Headers).To(HaveLen(1))
				Expect(iter.Range().Headers[0].Start).To(Equal(telem.TimeStamp(100 * telem.Second)))
				Expect(iter.Range().Range()).To(Equal(iter.View()))

				By("Reversing over a span")
				Expect(iter.PrevSpan(25 * telem.Second)).To(BeTrue())
				Expect(iter.View()).To(Equal(telem.TimeRange{
					Start: telem.TimeStamp(75 * telem.Second),
					End:   telem.TimeStamp(100 * telem.Second),
				}))
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.Ranges()).To(HaveLen(3))
				Expect(iter.Ranges()[0].Headers).To(HaveLen(1))
				Expect(iter.Ranges()[1].Headers).To(HaveLen(1))
				Expect(iter.Range().Range()).To(Equal(iter.View()))

				By("Reversing over a span again")
				Expect(iter.PrevSpan(25 * telem.Second)).To(BeTrue())
				Expect(iter.View()).To(Equal(telem.TimeRange{
					Start: telem.TimeStamp(50 * telem.Second),
					End:   telem.TimeStamp(75 * telem.Second),
				}))
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.Ranges()).To(HaveLen(3))
				Expect(iter.Ranges()[0].Headers).To(HaveLen(1))
				Expect(iter.Ranges()[1].Headers).To(HaveLen(1))
				Expect(iter.Range().Headers[0].Start).To(Equal(telem.TimeStamp(0)))
				Expect(iter.Range().Range()).To(Equal(iter.View()))

				By("Reversing over the global range boundary")
				Expect(iter.PrevSpan(100 * telem.Second)).To(BeTrue())
				Expect(iter.View()).To(Equal(telem.TimeRange{
					Start: telem.TimeStamp(0 * telem.Second),
					End:   telem.TimeStamp(50 * telem.Second),
				}))
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.Ranges()).To(HaveLen(3))
				Expect(iter.Ranges()[0].Headers).To(HaveLen(1))
				Expect(iter.Ranges()[1].Headers).To(HaveLen(1))

				By("Reversing over the global range boundary again")
				Expect(iter.PrevSpan(100 * telem.Second)).To(BeFalse())
				Expect(iter.Valid()).To(BeFalse())

				By("Moving forward over the next span")
				Expect(iter.NextSpan(100 * telem.Second)).To(BeTrue())
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.View()).To(Equal(telem.TimeRange{
					Start: telem.TimeStamp(50 * telem.Second),
					End:   telem.TimeStamp(150 * telem.Second),
				}))
				Expect(iter.Ranges()).To(HaveLen(3))
				Expect(iter.Ranges()[0].Headers).To(HaveLen(2))
				Expect(iter.Ranges()[2].Headers).To(HaveLen(2))

				By("Seeking over a range")
				nextRange := telem.TimeRange{
					Start: 0,
					End:   telem.TimeStamp(50 * telem.Second),
				}
				Expect(iter.SetRange(nextRange)).To(BeTrue())
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.View()).To(Equal(nextRange))
				Expect(iter.Range().Range()).To(Equal(nextRange))
				Expect(iter.Close()).To(Succeed())
			})

		})

	})

})
