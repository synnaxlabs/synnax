// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Persist only claim check", func() {
	It("Should release the delete hold of a persist only writer on a virtual channel",
		func(ctx SpecContext) {
			db := mustOpenDBOnFS(ctx, fs.NewMem())
			key := GenerateChannelKey()
			Expect(db.CreateChannel(ctx, virtualChannel(key, "cmd"))).To(Succeed())
			keys := channel.Keys{channel.Key(key)}
			w := MustSucceed(db.svc.Open(context.WithoutCancel(ctx), writer.Config{
				ControlSubject: xcontrol.Subject{Key: "a", Name: "a"},
				Keys:           keys,
				Start:          10 * telem.SecondTS,
				Mode:           writer.ModePersistOnly,
				Sync:           new(true),
			}))
			Expect(w.Close()).To(Succeed())
			release := MustSucceed(db.svc.ClaimDelete(keys))
			release()
		},
	)
})
