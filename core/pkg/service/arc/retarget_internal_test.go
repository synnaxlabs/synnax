// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

var _ = ginkgo.Describe("Channel retarget", func() {
	reads := func(b *Base) *[]channel.Key { return &b.Reads }
	params := func() channelParams {
		return channelParams{"on": types.Params{
			{Name: "channel", Type: types.Chan(types.F64())},
		}}
	}
	retarget := retargetChannel(types.ChanDirectionRead, reads, params)

	ginkgo.It("Should move a graph node's input and the reads to the key", func() {
		g := graph.Graph{
			Nodes: graph.Nodes{{Key: "source"}},
			Inputs: map[string]msgpack.EncodedJSON{
				"source": {"type": "on", "channel": float64(1)},
			},
		}
		a := Arc{Variant: AutomationArc{Base: Base{
			Source: Source{Variant: GraphSource{Graph: g}},
			Reads:  []channel.Key{1, 3},
		}}}
		retarget(&a, 1, 2)
		b := a.Base()
		gomega.Expect(b.Reads).To(gomega.Equal([]channel.Key{2, 3}))
		src, ok := b.Source.Variant.(GraphSource)
		gomega.Expect(ok).To(gomega.BeTrue())
		gomega.Expect(src.Graph.Inputs["source"]["channel"]).
			To(gomega.Equal(float64(2)))
	})

	ginkgo.It("Should keep a text Arc's text as it is", func() {
		raw := "ck_in * 2.0 -> ck_out"
		a := Arc{Variant: AutomationArc{Base: Base{
			Source: Source{Variant: TextSource{Text: text.Text{Raw: raw}}},
			Reads:  []channel.Key{1},
		}}}
		retarget(&a, 1, 2)
		b := a.Base()
		gomega.Expect(b.Reads).To(gomega.Equal([]channel.Key{2}))
		src, ok := b.Source.Variant.(TextSource)
		gomega.Expect(ok).To(gomega.BeTrue())
		gomega.Expect(src.Text.Raw).To(gomega.Equal(raw))
	})
})
