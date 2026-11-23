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
	"context"
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	channelv1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// compileFailures returns the description of each calculation compile failure status.
func compileFailures(ctx context.Context, n testutil.Node) []string {
	GinkgoHelper()
	var statuses []status.Status[any]
	Expect(gorp.NewRetrieve[status.Key, status.Status[any]]().
		Where(gorp.Match(func(_ gorp.Context, s *status.Status[any]) (bool, error) {
			return s.Message == "Failed to compile calculation", nil
		})).
		Entries(&statuses).
		Exec(ctx, n.DB)).To(Succeed())
	out := make([]string, len(statuses))
	for i, s := range statuses {
		out[i] = s.Description
	}
	return out
}

var _ = Describe("Dashed name upgrade breaker", func() {
	It("Should compile a v0.62 calculation that names dashed channels", func(
		ctx SpecContext,
	) {
		n := testutil.NewNode(ctx)
		stored := storeV062(ctx, n, []string{"pt-01", "pt-02", "delta"}, func(
			cs []channelv1.Channel,
		) {
			cs[2].Expression = "return pt-01 - pt-02"
		})
		l := MustOpen(svctestutil.OpenLayer(ctx, n))
		c := calculationOf(ctx, n, l.Calculation, stored[2].Key())
		Expect(c.Expression).To(Equal("return 'pt-01' - 'pt-02'"))
		Expect(c.Reads).To(ConsistOf(stored[0].Key(), stored[1].Key()),
			"compile failures: %v", compileFailures(ctx, n))
	})

	DescribeTable("Should compile a v0.62 Arc that names dashed channels",
		func(ctx SpecContext, raw string) {
			n := testutil.NewNode(ctx)
			storeV062(ctx, n, []string{"pt-01", "pt-02"}, func([]channelv1.Channel) {})
			keys := storeArcs(ctx, n, []string{raw}, nil)
			l := MustOpen(svctestutil.OpenLayer(ctx, n))
			_, err := l.Arc.CompileProgram(ctx, keys[0])
			Expect(err).ToNot(HaveOccurred(), "text %q", sourceOf(ctx, n, keys[0]))
		},
		Entry("a flow",
			"func scale() {v f32} f32 { return v * 2.0 }\npt-01 -> scale() -> pt-02"),
		Entry("a flow with no spaces",
			"func scale() {v f32} f32 { return v * 2.0 }\npt-01->scale()->pt-02"),
		Entry("an authority entry",
			"authority (pt-02 200)\nfunc scale() {v f32} f32 { return v * 2.0 }\n"+
				"pt-01 -> scale() -> pt-02"),
	)

	It(
		"Should compile a v0.62 Arc that names a dashed channel in a format string",
		func(
			ctx SpecContext,
		) {
			n := testutil.NewNode(ctx)
			storeV062(
				ctx,
				n,
				[]string{"pt-01", "pt-02", "msg"},
				func(cs []channelv1.Channel) {
					cs[2].DataType = telem.StringT
				},
			)
			keys := storeArcs(ctx, n, []string{
				"func label() {v f32} str { return f\"{v:.2f} {pt-02:.2f}\" }\n" +
					"pt-01 -> label() -> msg",
			}, nil)
			l := MustOpen(svctestutil.OpenLayer(ctx, n))
			_, err := l.Arc.CompileProgram(ctx, keys[0])
			Expect(err).ToNot(HaveOccurred(), "text %q", sourceOf(ctx, n, keys[0]))
		},
	)

	It("Should read the dashed channel that a v0.62 Arc export names", func(
		ctx SpecContext,
	) {
		n := testutil.NewNode(ctx)
		storeV062(ctx, n, []string{"pt", "pt-01", "out"}, func([]channelv1.Channel) {})
		l := MustOpen(svctestutil.OpenLayer(ctx, n))
		// The export of an Arc from a v0.62 Core that ran with dashed names on.
		var env imex.Envelope
		Expect(json.Unmarshal([]byte(`{
			"version": 4,
			"type": "arc",
			"key": "11111111-2222-3333-4444-555555555555",
			"name": "v062 export",
			"source": {"mode": "text", "text": {"raw": "pt-01 * 2.0 -> out"}}
		}`), &env)).To(Succeed())
		var id resource.ID
		Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) (err error) {
			id, err = l.ImEx.Import(ctx, tx, env, imex.ImportOptions{
				Parent: resource.ID{Type: resource.TypeProject, Key: "parent"},
			})
			return err
		})).To(Succeed())
		key := MustSucceed(uuid.Parse(id.Key))
		_, compileErr := l.Arc.CompileProgram(ctx, key)
		Expect(sourceOf(ctx, n, key)).To(Equal("'pt-01' * 2.0 -> out"),
			"compile error: %v", compileErr)
	})

	It("Should leave a dashed run of a v0.62 Arc export that no channel holds", func(
		ctx SpecContext,
	) {
		n := testutil.NewNode(ctx)
		storeV062(ctx, n, []string{"pt", "out"}, func([]channelv1.Channel) {})
		l := MustOpen(svctestutil.OpenLayer(ctx, n))
		var env imex.Envelope
		Expect(json.Unmarshal([]byte(`{
			"version": 4,
			"type": "arc",
			"key": "11111111-2222-3333-4444-666666666666",
			"name": "v062 export",
			"source": {"mode": "text", "text": {"raw": "pt-01 * 2.0 -> out"}}
		}`), &env)).To(Succeed())
		var id resource.ID
		Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) (err error) {
			id, err = l.ImEx.Import(ctx, tx, env, imex.ImportOptions{
				Parent: resource.ID{Type: resource.TypeProject, Key: "parent"},
			})
			return err
		})).To(Succeed())
		Expect(sourceOf(ctx, n, MustSucceed(uuid.Parse(id.Key)))).
			To(Equal("pt-01 * 2.0 -> out"))
	})
})
