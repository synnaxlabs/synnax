// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Import of graph inputs that hold text check", func() {
	readAll := func(resource.ID) bool { return true }

	DescribeTable("Should import a graph input that holds text as stored", func(
		ctx SpecContext,
		stored any,
	) {
		a := newAutomation(arc.Base{
			Name: "graph_input_" + uuid.New().String()[:8],
			Source: arc.Source{Variant: arc.GraphSource{Graph: graph.Graph{
				Nodes: graph.Nodes{{Key: "n"}},
				Inputs: map[string]msgpack.EncodedJSON{
					"n": {"type": "on", "channel": stored},
				},
			}}},
		})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &a)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, a.Base().Key)
			})).To(Succeed())
		})
		env := MustSucceed(svc.Export(ctx, db, arc.ResourceID(a.Base().Key)))
		file := MustSucceed(json.Marshal(env.WithEncodeOptions(
			channelSvc.LinkWriter(ctx, db, readAll),
		)))
		var decoded imex.Envelope
		Expect(imex.Codec.Decode(file, &decoded)).To(Succeed())
		decoded.Name = "imported_" + uuid.New().String()[:8]
		links := channelSvc.NewLinkReader(ctx, db, readAll)
		opts := decoded.WithDecodeOptions(links.Options())
		write := MustSucceed(svc.Decode(ctx, opts))
		var id resource.ID
		Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
			id, err = write(ctx, tx, imex.ImportOptions{})
			return err
		})).To(Succeed())
		Expect(links.Err()).To(Succeed())
		k := MustSucceed(uuid.Parse(id.Key))
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, k)
			})).To(Succeed())
		})
		var res arc.Arc
		Expect(svc.NewRetrieve().Where(arc.MatchKeys(k)).Entry(&res).Exec(ctx, db)).
			To(Succeed())
		src, ok := res.Base().Source.Variant.(arc.GraphSource)
		Expect(ok).To(BeTrue())
		Expect(src.Graph.Inputs["n"]).To(HaveKeyWithValue("channel", stored))
	},
		Entry("a key as text", "1048577"),
		Entry("a name that no channel holds", "no_such_channel_name"),
	)
})
