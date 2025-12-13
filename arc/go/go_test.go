package arc_test

import (
	"fmt"
	"io/fs"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Arc", func() {
	It("Should compile an entire calc.arc file", func() {
		testDataDir := os.DirFS("./testdata")
		f := MustSucceed(fs.ReadFile(testDataDir, "calc.arc"))
		Expect(f).ToNot(BeEmpty())
		t := arc.Text{Raw: string(f)}
		Expect(t.Raw).ToNot(BeEmpty())
		mod := MustSucceed(arc.CompileText(ctx, t, arc.WithResolver(symbol.MapResolver{
			"ox_pt_1": arc.Symbol{
				Name: "ox_pt_1",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F32()),
				ID:   1,
			},
			"ox_pt_doubled": arc.Symbol{
				Name: "ox_pt_doubled",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F32()),
				ID:   2,
			},
		})))

		fmt.Println(mod)

		// Verify correct node structure
		Expect(mod.Nodes).To(HaveLen(3))

		// First node: source channel (on)
		Expect(mod.Nodes[0].Type).To(Equal("on"))
		Expect(mod.Nodes[0].Channels.Read.Contains(uint32(1))).To(BeTrue())
		Expect(mod.Nodes[0].Outputs).To(HaveLen(1))

		// Second node: calc function
		Expect(mod.Nodes[1].Type).To(Equal("calc"))

		// Third node: sink channel (write)
		Expect(mod.Nodes[2].Type).To(Equal("write"))
		Expect(mod.Nodes[2].Channels.Write.Contains(uint32(2))).To(BeTrue())
		Expect(mod.Nodes[2].Inputs).To(HaveLen(1))

		Expect(mod.Edges).To(HaveLen(2))
	})
})
