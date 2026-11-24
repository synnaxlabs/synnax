package v14

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	symbolv4 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v4"
	symbol "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v5"
	xcolor "github.com/synnaxlabs/x/color/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	zBlue   = xcolor.Color{R: 40, G: 80, B: 220, A: 255}
	zGreen  = xcolor.Color{R: 40, G: 200, B: 90, A: 255}
	zRed    = xcolor.Color{R: 220, G: 40, B: 40, A: 255}
	zYellow = xcolor.Color{R: 220, G: 200, B: 40, A: 255}
	zWhite  = xcolor.Color{R: 255, G: 255, B: 255, A: 255}
	zBlack  = xcolor.Color{A: 255}
)

func zr(key string, stroke, fill *xcolor.Color) symbolv4.Region {
	return symbolv4.Region{Key: key, Name: key, Selectors: []string{"." + key}, StrokeColor: stroke, FillColor: fill}
}

func zc(c xcolor.Color) *xcolor.Color { return &c }

func zspec(states ...[]symbolv4.Region) symbolv4.Spec {
	s := symbolv4.Spec{
		Variant: "actuator",
		SVG:     `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect class="a"/><rect class="b"/><rect class="c"/></svg>`,
	}
	for i, rs := range states {
		s.States = append(s.States, symbolv4.State{Key: []string{"off", "on", "x"}[i], Name: []string{"Off", "On", "X"}[i], Regions: rs})
	}
	return s
}

func zstates(states ...[]symbolv4.Region) []symbolv4.State { return zspec(states...).States }

var _ = Describe("zz agreement", func() {
	vent := zspec(
		[]symbolv4.Region{zr("a", zc(zBlue), zc(zBlue))},
		[]symbolv4.Region{zr("a", zc(zGreen), zc(zGreen))},
	)
	acb := zspec(
		[]symbolv4.Region{zr("a", zc(zBlue), zc(zBlue)), zr("c", zc(zBlue), zc(zBlue)), zr("b", zc(zBlack), nil)},
		[]symbolv4.Region{zr("c", zc(zGreen), zc(zGreen)), zr("a", zc(zGreen), zc(zGreen)), zr("b", zc(zBlack), nil)},
	)
	whiteFill := zspec(
		[]symbolv4.Region{zr("a", zc(zBlue), zc(zWhite))},
		[]symbolv4.Region{zr("a", zc(zGreen), zc(zWhite))},
	)
	DescribeTable("agree", func(ctx SpecContext, src symbolv4.Spec, states []symbolv4.State, wantCopy bool) {
		e4, c4, err := redraw(ctx, src, states)
		Expect(err).ToNot(HaveOccurred())
		Expect(c4 != nil).To(Equal(wantCopy))
		src5, entries, err := symbol.MigrateStoredSpec(ctx, src)
		Expect(err).ToNot(HaveOccurred())
		e5, c5, err := redrawStored(src5, entries, states)
		Expect(err).ToNot(HaveOccurred())
		if c4 == nil {
			Expect(c5).To(BeNil())
			Expect(e5).To(Equal(e4))
			return
		}
		Expect(c5).ToNot(BeNil())
		cs, ce := MustSucceed2(symbol.MigrateStoredSpec(ctx, *c4))
		GinkgoWriter.Printf("upgrade %v\nimport  %v\n", draw(cs, ce), draw(*c5, e5))
		Expect(draw(cs, ce).equal(draw(*c5, e5))).To(BeTrue())
	},
		Entry("equal", vent, vent.States, false),
		Entry("entry colors", vent, zstates([]symbolv4.Region{zr("a", zc(zRed), zc(zRed))}, []symbolv4.Region{zr("a", zc(zYellow), zc(zYellow))}), false),
		Entry("first state", vent, zstates([]symbolv4.Region{zr("a", zc(zRed), zc(zRed))}), false),
		Entry("white fill", vent, zstates([]symbolv4.Region{zr("a", zc(zRed), zc(zWhite))}, []symbolv4.Region{zr("a", zc(zYellow), zc(zWhite))}), true),
		Entry("acb recolor", acb, zstates(
			[]symbolv4.Region{zr("a", zc(zRed), zc(zRed)), zr("c", zc(zRed), zc(zRed)), zr("b", zc(zBlack), nil)},
			[]symbolv4.Region{zr("c", zc(zYellow), zc(zYellow)), zr("a", zc(zYellow), zc(zYellow)), zr("b", zc(zBlack), nil)},
		), false),
		Entry("acb white A fill", acb, zstates(
			[]symbolv4.Region{zr("a", zc(zBlue), zc(zWhite)), zr("c", zc(zBlue), zc(zBlue)), zr("b", zc(zBlack), nil)},
			[]symbolv4.Region{zr("c", zc(zGreen), zc(zGreen)), zr("a", zc(zGreen), zc(zWhite)), zr("b", zc(zBlack), nil)},
		), true),
		Entry("fill joins the entries", whiteFill, zstates(
			[]symbolv4.Region{zr("a", zc(zRed), zc(zRed))},
			[]symbolv4.Region{zr("a", zc(zYellow), zc(zYellow))},
		), true),
		Entry("stale selectors", vent, func() []symbolv4.State {
			st := zstates([]symbolv4.Region{zr("a", zc(zRed), zc(zRed))}, []symbolv4.Region{zr("a", zc(zYellow), zc(zYellow))})
			st[0].Regions[0].Selectors = []string{".b"}
			st[1].Regions[0].Selectors = []string{".b"}
			return st
		}(), true),
		Entry("color removed", vent, zstates([]symbolv4.Region{zr("a", zc(zRed), nil)}, []symbolv4.Region{zr("a", zc(zYellow), nil)}), true),
	)
})
