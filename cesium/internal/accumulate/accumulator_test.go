package accumulate_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/accumulate"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Accumulator", func() {
	Context("Single Segment", func() {
		Context("Within Bounds", func() {
			It("Should accumulate the segment correctly", func() {
				a := &accumulate.Accumulator{Density: 1, Bounds: position.Range{End: 100}}
				Expect(a.Accumulate(core.SegmentMD{
					Alignment: 0,
					Size:      100,
					Offset:    0,
				})).To(BeTrue())
				Expect(a.Segments).To(HaveLen(1))
				Expect(a.Segments[0].Alignment).To(Equal(position.Position(0)))
				Expect(a.Segments[0].Size).To(Equal(telem.Size(100)))
				Expect(a.Segments[0].Offset).To(Equal(telem.Offset(0)))
				Expect(a.Satisfied()).To(BeTrue())
				Expect(a.PartiallySatisfied()).To(BeTrue())
			})
		})
		Context("After Bounds", func() {
			It("Should not accumulate the segment", func() {
				a := &accumulate.Accumulator{Density: 1, Bounds: position.Range{Start: 100, End: 200}}
				Expect(a.Accumulate(core.SegmentMD{
					Alignment: 0,
					Size:      100,
					Offset:    0,
				})).To(BeFalse())
				Expect(a.Segments).To(HaveLen(0))
				Expect(a.Satisfied()).To(BeFalse())
				Expect(a.PartiallySatisfied()).To(BeFalse())
			})
		})
		Context("Overlaps with Beginning of Bounds", func() {
			It("Should accumulate the segment correctly", func() {
				a := &accumulate.Accumulator{Density: 1, Bounds: position.Range{Start: 50, End: 200}}
				Expect(a.Accumulate(core.SegmentMD{
					Alignment: 0,
					Size:      100,
					Offset:    0,
				})).To(BeTrue())
				Expect(a.Segments).To(HaveLen(1))
				Expect(a.Segments[0].Alignment).To(Equal(position.Position(50)))
				Expect(a.Segments[0].Size).To(Equal(telem.Size(50)))
				Expect(a.Segments[0].Offset).To(Equal(telem.Offset(50)))
				Expect(a.Satisfied()).To(BeFalse())
				Expect(a.PartiallySatisfied()).To(BeTrue())
			})
		})
		Context("Overlaps with End of Bounds", func() {
			It("Should accumulate the segment correctly", func() {
				a := &accumulate.Accumulator{Density: 1, Bounds: position.Range{Start: 0, End: 50}}
				Expect(a.Accumulate(core.SegmentMD{
					Alignment: 0,
					Size:      100,
					Offset:    0,
				})).To(BeTrue())
				Expect(a.Segments).To(HaveLen(1))
				Expect(a.Segments[0].Alignment).To(Equal(position.Position(0)))
				Expect(a.Segments[0].Size).To(Equal(telem.Size(50)))
				Expect(a.Segments[0].Offset).To(Equal(telem.Offset(0)))
				Expect(a.Satisfied()).To(BeTrue())
				Expect(a.PartiallySatisfied()).To(BeTrue())
			})
		})
	})
})
