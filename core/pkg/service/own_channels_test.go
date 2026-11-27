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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channelv1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("The Core's own channels", func() {
	// storeSpread makes n hold a v0.62 store in which two channels in two folders, and
	// none at the top level, hold name. It returns their keys in key order.
	storeSpread := func(
		ctx SpecContext,
		n testutil.Node,
		name string,
		dataType telem.DataType,
	) (channel.Key, channel.Key) {
		GinkgoHelper()
		a, b := folder("Stand A"), folder("Stand B")
		stored := storeV062(ctx, n, []string{name, name}, func(cs []channelv1.Channel) {
			cs[0].Group, cs[1].Group = &a.Key, &b.Key
			cs[0].DataType, cs[1].DataType = dataType, dataType
		}, a, b)
		lower, higher := stored[0].Key(), stored[1].Key()
		Expect(higher).To(BeNumerically(">", lower))
		return lower, higher
	}
	// expectTaken expects a warning status of name that names other and uses taken.
	expectTaken := func(
		ctx SpecContext,
		n testutil.Node,
		name string,
		other, taken channel.Key,
	) {
		GinkgoHelper()
		var statuses []status.Status[any]
		Expect(gorp.NewRetrieve[status.Key, status.Status[any]]().
			Where(gorp.Match(func(_ gorp.Context, s *status.Status[any]) (bool, error) {
				return s.Name == name, nil
			})).
			Entries(&statuses).
			Exec(ctx, n.DB)).To(Succeed())
		Expect(statuses).To(ContainElement(SatisfyAll(
			HaveField("Variant", status.VariantWarning),
			HaveField("Description", SatisfyAll(
				ContainSubstring(other.String()),
				ContainSubstring(fmt.Sprintf("uses the one with key %s", taken)),
			)),
		)))
	}

	It("Should send task commands on the highest key of the command channel's name "+
		"when only folders hold it, and warn", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		lower, higher := storeSpread(ctx, n, "sy_task_cmd", telem.JSONT)
		l := MustOpen(svctestutil.OpenLayer(ctx, n))
		Expect(l.Task.CommandChannelKey()).To(Equal(higher))
		expectTaken(ctx, n, "sy_task_cmd", lower, higher)
	})

	DescribeTable("Should open on the highest key of the name of a channel that the "+
		"Core makes when only folders hold it, and warn", func(
		ctx SpecContext,
		name string,
		dataType telem.DataType,
	) {
		n := testutil.NewNode(ctx)
		lower, higher := storeSpread(ctx, n, name, dataType)
		MustOpen(svctestutil.OpenLayer(ctx, n))
		expectTaken(ctx, n, name, lower, higher)
	},
		Entry("a signals channel", "sy_task_set", telem.JSONT),
		Entry("a metrics channel", "sy_node_1_metrics_cpu_percentage", telem.Float32T),
	)
})
