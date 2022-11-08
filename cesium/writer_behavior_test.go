package cesium_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer Behavior", Ordered, func() {
	var (
		db cesium.DB
	)
	BeforeAll(func() {
		db = openMemDB()
	})
	AfterAll(func() {
		Expect(db.Close()).To(Succeed())
	})
	Describe("Write", func() {
		Specify("Uncommitted writes are invisible to readers", func() {
			chs, w := createWriter(
				db,
				cesium.Channel{Rate: 100 * telem.Hz, Density: telem.Bit64},
			)
			key := chs[0].Key
			Expect(w.Write([]cesium.Segment{
				{
					ChannelKey: key,
					Start:      10 * telem.SecondTS,
					Data:       Marshal([]float64{1, 2, 3}),
				},
			})).To(BeTrue())
			Expect(w.Close()).To(Succeed())
			d, err := db.Read(telem.TimeRangeMax, key)
			Expect(err).ToNot(HaveOccurred())
			Expect(d).To(HaveLen(0))
		})
		Specify("Commits should fail after an invalid write", func() {
			chs, w := createWriter(
				db,
				cesium.Channel{Rate: 100 * telem.Hz, Density: telem.Bit64},
			)
			key := chs[0].Key
			Expect(w.Write([]cesium.Segment{
				{
					ChannelKey: key,
					Start:      10 * telem.SecondTS,
					Data:       Marshal([]float64{1, 2, 3}),
				},
				{
					ChannelKey: key,
					Start:      10 * telem.SecondTS,
					Data:       Marshal([]float64{1, 2, 3}),
				},
			})).To(BeTrue())
			Expect(w.Commit()).To(BeFalse())
			Expect(w.Close()).To(HaveOccurredAs(validate.Error))
			d, err := db.Read(telem.TimeRangeMax, key)
			Expect(err).ToNot(HaveOccurred())
			Expect(d).To(HaveLen(0))
		})
		Specify("Writes should succeed after the user has acknowledged the error", func() {
			chs, w := createWriter(
				db,
				cesium.Channel{Rate: 100 * telem.Hz, Density: telem.Bit64},
			)
			key := chs[0].Key
			Expect(w.Write([]cesium.Segment{
				{

					ChannelKey: key,
					Start:      10 * telem.SecondTS,
					Data:       Marshal([]float64{1, 2, 3}),
				},
			})).To(BeTrue())
			Expect(w.Write([]cesium.Segment{

				{
					ChannelKey: key,
					Start:      10 * telem.SecondTS,
					Data:       Marshal([]float64{1, 2, 3}),
				},
			})).To(BeTrue())
			Expect(w.Commit()).To(BeFalse())
			Expect(w.Error()).To(HaveOccurredAs(validate.Error))
			Expect(w.Commit()).To(BeTrue())
			Expect(w.Close()).To(Succeed())
			d, err := db.Read(telem.TimeRangeMax, key)
			Expect(err).ToNot(HaveOccurred())
			Expect(d).To(HaveLen(1))
		})
	})
	Describe("Commit", func() {
		It("Should allow the user to commit writes several times", func() {
			chs, w := createWriter(
				db,
				cesium.Channel{Rate: 1 * telem.Hz, Density: telem.Bit64},
			)
			key := chs[0].Key
			Expect(w.Write([]cesium.Segment{
				{
					ChannelKey: key,
					Start:      10 * telem.SecondTS,
					Data:       Marshal([]float64{1, 2, 3}),
				},
			})).To(BeTrue())
			Expect(w.Commit()).To(BeTrue())
			Expect(w.Write([]cesium.Segment{
				{
					ChannelKey: key,
					Start:      13 * telem.SecondTS,
					Data:       Marshal([]float64{1, 2, 3}),
				},
			})).To(BeTrue())
			Expect(w.Commit()).To(BeTrue())
			Expect(w.Close()).To(Succeed())
			d, err := db.Read(telem.TimeRangeMax, key)
			Expect(err).ToNot(HaveOccurred())
			Expect(d).To(HaveLen(2))
		})
	})
})
