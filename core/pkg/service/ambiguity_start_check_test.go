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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	channelv1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// A v0.62 Core with name validation off can hold a user channel that shares the name
// of a channel the Core creates at each start, in another folder. D-448 keeps both
// names, so every start after the upgrade must still open.
var _ = Describe("Core channels at start (check)", func() {
	DescribeTable(
		"Should open a v0.62 store whose user channel in a folder shares the name of a "+
			"channel the Core creates at start",
		func(ctx SpecContext, name string, dataType telem.DataType) {
			n := testutil.NewNode(ctx)
			rigs := folder("Rigs")
			storeV062(
				ctx,
				n,
				[]string{name, name},
				func(cs []channelv1.Channel) {
					// The Core's channel, as v0.62 created it.
					cs[0].Internal = true
					cs[0].DataType = dataType
					// The user's channel of the same name in a folder.
					cs[1].Group = &rigs.Key
				},
				rigs,
			)
			l := MustSucceed(svctestutil.OpenLayer(ctx, n))
			Expect(l.Close()).To(Succeed())
		},
		Entry("the task command channel", "sy_task_cmd", telem.JSONT),
		Entry("a signals channel", "sy_channel_action_set", telem.JSONT),
	)
})
