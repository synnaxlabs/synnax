// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channels_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/stl/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Body reads of a uuid series", func() {
	It("Should not invent text from the bytes of a uuid series", func(ctx SpecContext) {
		rt := testutil.NewRuntime(ctx)
		defer func() { Expect(rt.Close(ctx)).To(Succeed()) }()
		cs := channels.NewProgramState([]channels.Digest{
			{Key: 3, Name: "label", DataType: telem.StringT},
		})
		ss := strings.NewProgramState()
		MustSucceed(channels.NewHost(ctx, rt.Underlying(), cs, ss))
		rt.Passthrough(ctx, "channels")
		// The 16 bytes of the uuid read as two length-prefixed strings: "xxxxxx", then
		// "hi".
		id := []byte{6, 0, 0, 0, 'x', 'x', 'x', 'x', 'x', 'x', 2, 0, 0, 0, 'h', 'i'}
		text := fmt.Sprintf(
			"%x-%x-%x-%x-%x",
			id[:4], id[4:6], id[6:8], id[8:10], id[10:],
		)
		cs.Ingest(telem.UnaryFrame[uint32](3, telem.Series{
			DataType: telem.UUIDT,
			Data:     id,
		}))
		var read string
		failed := func() (failed bool) {
			defer func() { failed = recover() != nil }()
			result := rt.Call(ctx, "channels", "read_str", testutil.U32(3))
			read, _ = ss.Get(testutil.AsU32(result[0]))
			return false
		}()
		if !failed {
			Expect(read).To(Equal(text))
		}
	})
})
