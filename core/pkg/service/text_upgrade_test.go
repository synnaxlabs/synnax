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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	svcarc "github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	channelv1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

// storeGraph stores an Arc whose graph holds fns, and returns its key.
func storeGraph(ctx context.Context, n testutil.Node, fns ir.Functions) svcarc.Key {
	GinkgoHelper()
	Expect(MustSucceed(svcarc.OpenTable(ctx, svcarc.TableConfig{DB: n.DB})).Close()).
		To(Succeed())
	isRoot := func(_ gorp.Context, d *directory.Directory) (bool, error) {
		return d.Parent == nil, nil
	}
	var root directory.Directory
	Expect(gorp.NewRetrieve[directory.Key, directory.Directory]().
		Where(gorp.Match(isRoot)).
		Entry(&root).
		Exec(ctx, n.DB)).To(Succeed())
	a := svcarc.Arc{Variant: svcarc.AutomationArc{Base: svcarc.Base{
		Key:       uuid.New(),
		Name:      "graph",
		Directory: &root.Key,
		Source: svcarc.Source{Variant: svcarc.GraphSource{Graph: graph.Graph{
			Functions: fns,
		}}},
	}}}
	Expect(gorp.WrapWriter[svcarc.Key, svcarc.Arc](n.DB).Set(ctx, a)).To(Succeed())
	return a.Base().Key
}

// bodiesOf returns the body of each graph function of the Arc with key.
func bodiesOf(ctx context.Context, n testutil.Node, key svcarc.Key) []string {
	GinkgoHelper()
	var a svcarc.Arc
	Expect(gorp.NewRetrieve[svcarc.Key, svcarc.Arc]().
		Where(gorp.MatchKeys[svcarc.Key, svcarc.Arc](key)).
		Entry(&a).
		Exec(ctx, n.DB)).To(Succeed())
	src, ok := a.Base().Source.Variant.(svcarc.GraphSource)
	Expect(ok).To(BeTrue())
	bodies := make([]string, len(src.Graph.Functions))
	for i, f := range src.Graph.Functions {
		bodies[i] = f.Body.Raw
	}
	return bodies
}

// keptStatuses returns each status that reports an old name kept in text.
func keptStatuses(ctx context.Context, n testutil.Node) []status.Status[any] {
	GinkgoHelper()
	isKept := func(_ gorp.Context, s *status.Status[any]) (bool, error) {
		return s.Message == "Kept an old channel name in text", nil
	}
	var statuses []status.Status[any]
	Expect(gorp.NewRetrieve[status.Key, status.Status[any]]().
		Where(gorp.Match(isKept)).
		Entries(&statuses).
		Exec(ctx, n.DB)).To(Succeed())
	return statuses
}

var _ = Describe("Text upgrade", func() {
	// storeMetrics stores a v0.62 channel named Metrics beside the Metrics folder, so
	// the upgrade renames it to Metrics_2.
	storeMetrics := func(ctx context.Context, n testutil.Node) {
		GinkgoHelper()
		storeV062(
			ctx,
			n,
			[]string{groupv1.MetricsFolder},
			func([]channelv1.Channel) {},
			folder(groupv1.MetricsFolder),
		)
	}

	It("Should report the kept reference of each graph function when two bodies keep "+
		"one at the same place", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		storeMetrics(ctx, n)
		storeGraph(ctx, n, ir.Functions{
			{Key: "a", Body: ir.Body{Raw: "{ return Metrics * }"}},
			{Key: "b", Body: ir.Body{Raw: "{ return Metrics * }"}},
		})
		MustOpen(svctestutil.OpenLayer(ctx, n))
		Expect(keptStatuses(ctx, n)).To(HaveLen(2))
	})

	It("Should keep a graph body when a sibling function holds the new name, as "+
		"v0.62 bound each body with every function of the graph in scope", func(
		ctx SpecContext,
	) {
		n := testutil.NewNode(ctx)
		storeMetrics(ctx, n)
		key := storeGraph(ctx, n, ir.Functions{
			{Key: "Metrics_2", Body: ir.Body{Raw: "{ return 1.0 }"}},
			{Key: "f", Body: ir.Body{Raw: "{ return Metrics * 2 }"}},
		})
		MustOpen(svctestutil.OpenLayer(ctx, n))
		Expect(bodiesOf(ctx, n, key)[1]).To(Equal("{ return Metrics * 2 }"))
		Expect(keptStatuses(ctx, n)).To(ConsistOf(HaveField(
			"Description",
			`The upgrade renamed channel "Metrics" to "Metrics_2", but the text of `+
				`function "f" of Arc "graph" still reads the old name at line 1, `+
				`column 10, because a symbol that the text declares holds one of `+
				`these names there. Edit the text so that it reads the channel.`,
		)))
	})

	It("Should report a kept reference of an Arc by the name of the Arc", func(
		ctx SpecContext,
	) {
		n := testutil.NewNode(ctx)
		storeMetrics(ctx, n)
		storeArcs(ctx, n, []string{"Metrics_2 := 1.0\nx := Metrics"}, nil)
		MustOpen(svctestutil.OpenLayer(ctx, n))
		Expect(keptStatuses(ctx, n)).To(ConsistOf(HaveField(
			"Description",
			`The upgrade renamed channel "Metrics" to "Metrics_2", but the text of `+
				`Arc "text" still reads the old name at line 2, column 6, because a `+
				`symbol that the text declares holds one of these names there. Edit `+
				`the text so that it reads the channel.`,
		)))
	})

	DescribeTable("Should write the new name into a graph body beside a function "+
		"that has no body yet", func(ctx SpecContext, fns ir.Functions, want string) {
		n := testutil.NewNode(ctx)
		storeMetrics(ctx, n)
		key := storeGraph(ctx, n, fns)
		MustOpen(svctestutil.OpenLayer(ctx, n))
		Expect(bodiesOf(ctx, n, key)[1]).To(Equal(want))
		Expect(keptStatuses(ctx, n)).To(BeEmpty())
	},
		Entry("a later input named like that function", ir.Functions{
			{Key: "g"},
			{
				Key:     "f",
				Inputs:  types.Params{{Name: "g", Type: types.F64()}},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F64()}},
				Body:    ir.Body{Raw: "{ return Metrics * g }"},
			},
		}, "{ return Metrics_2 * g }"),
		Entry("a second function under that key", ir.Functions{
			{Key: "g"},
			{Key: "g", Body: ir.Body{Raw: "{ return Metrics }"}},
		}, "{ return Metrics_2 }"),
	)

	It("Should open a v0.62 store that holds a graph function whose input holds the "+
		"key of its function", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		storeMetrics(ctx, n)
		// v0.62 reported this graph as a diagnostic when it compiled it.
		key := storeGraph(ctx, n, ir.Functions{{
			Key:     "scale",
			Inputs:  types.Params{{Name: "scale", Type: types.F64()}},
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F64()}},
			Body:    ir.Body{Raw: "{ return Metrics * scale }"},
		}})
		MustOpen(svctestutil.OpenLayer(ctx, n))
		Expect(bodiesOf(ctx, n, key)).To(Equal([]string{"{ return Metrics * scale }"}))
		Expect(keptStatuses(ctx, n)).To(HaveLen(1))
	})
})
