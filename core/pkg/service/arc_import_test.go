// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strconv"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	channelv1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

// arcEnvelope returns the envelope of an exported Arc with src, stamped with version.
func arcEnvelope(version int, src arc.Source) imex.Envelope {
	GinkgoHelper()
	var fields map[string]jsontext.Value
	Expect(json.Unmarshal(MustSucceed(json.Marshal(
		arc.Base{Key: uuid.New(), Name: "export", Source: src},
	)), &fields)).To(Succeed())
	fields["version"] = jsontext.Value(strconv.Itoa(version))
	fields["type"] = jsontext.Value(`"arc"`)
	var env imex.Envelope
	Expect(json.Unmarshal(MustSucceed(json.Marshal(fields)), &env)).To(Succeed())
	return env
}

// importArc imports env through l for an importer that can retrieve what retrievable
// reports, and returns the key of the new Arc.
func importArc(
	ctx SpecContext,
	n testutil.Node,
	l *service.Layer,
	env imex.Envelope,
	retrievable func(resource.ID) bool,
) arc.Key {
	GinkgoHelper()
	var id resource.ID
	Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) (err error) {
		id, err = l.ImEx.Import(ctx, tx, env, imex.ImportOptions{
			Parent:      resource.ID{Type: resource.TypeProject, Key: "parent"},
			Retrievable: retrievable,
		})
		return err
	})).To(Succeed())
	return MustSucceed(uuid.Parse(id.Key))
}

var _ = Describe("Arc import", func() {
	It("Should leave a bare dashed run of a v5 Arc export that a channel holds", func(
		ctx SpecContext,
	) {
		n := testutil.NewNode(ctx)
		storeV062(ctx, n, []string{"pt-01", "out"}, func([]channelv1.Channel) {})
		l := MustOpen(svctestutil.OpenLayer(ctx, n))
		env := arcEnvelope(5, arc.Source{Variant: arc.TextSource{
			Text: text.Text{Raw: "pt-01 * 2.0 -> out"},
		}})
		Expect(sourceOf(ctx, n, importArc(ctx, n, l, env, nil))).
			To(Equal("pt-01 * 2.0 -> out"))
	})

	It("Should quote a dashed channel in a graph function of a v0.62 Arc export", func(
		ctx SpecContext,
	) {
		n := testutil.NewNode(ctx)
		storeV062(ctx, n, []string{"pt-01"}, func([]channelv1.Channel) {})
		l := MustOpen(svctestutil.OpenLayer(ctx, n))
		env := arcEnvelope(4, arc.Source{Variant: arc.GraphSource{Graph: graph.Graph{
			Functions: ir.Functions{
				{Key: "f", Body: ir.Body{Raw: "return pt-01 * 2.0"}},
			},
		}}})
		Expect(sourceOf(ctx, n, importArc(ctx, n, l, env, nil))).
			To(Equal("return 'pt-01' * 2.0"))
	})

	It(
		"Should leave bare a dashed name whose channel the importer cannot retrieve",
		func(
			ctx SpecContext,
		) {
			n := testutil.NewNode(ctx)
			stored := storeV062(
				ctx,
				n,
				[]string{"pt-01", "pt-02"},
				func([]channelv1.Channel) {},
			)
			l := MustOpen(svctestutil.OpenLayer(ctx, n))
			env := arcEnvelope(
				4,
				arc.Source{Variant: arc.GraphSource{Graph: graph.Graph{
					Functions: ir.Functions{
						{Key: "f", Body: ir.Body{Raw: "return pt-01 + pt-02"}},
					},
				}}},
			)
			hidden := stored[0].ResourceID()
			Expect(
				sourceOf(ctx, n, importArc(ctx, n, l, env, func(id resource.ID) bool {
					return id != hidden
				})),
			).To(Equal("return pt-01 + 'pt-02'"))
		},
	)
})
