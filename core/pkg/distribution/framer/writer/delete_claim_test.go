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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Delete claim", func() {
	It("Should refuse a delete of a virtual channel that a writer holds, and keep "+
		"the writer's control", func(ctx SpecContext) {
		db := mustOpenDBOnFS(ctx, fs.NewMem())
		key := GenerateChannelKey()
		Expect(db.CreateChannel(ctx, virtualChannel(key, "cmd"))).To(Succeed())
		keys := channel.Keys{channel.Key(key)}
		open := func(name string, auth xcontrol.Authority) *writer.Writer {
			GinkgoHelper()
			return MustSucceed(db.svc.Open(context.WithoutCancel(ctx), writer.Config{
				ControlSubject: xcontrol.Subject{Key: name, Name: name},
				Keys:           keys,
				Start:          10 * telem.SecondTS,
				Authorities:    []xcontrol.Authority{auth},
				Sync:           new(true),
			}))
		}
		a := open("a", 200)
		Expect(db.svc.ClaimDelete(keys)).Error().To(SatisfyAll(
			MatchError(writer.ErrChannelInUse),
			MatchError(ContainSubstring("1 unclosed writers")),
		))
		b := open("b", 100)
		Expect(b.Write(frame.NewUnary(keys[0], telem.NewSeriesV[int64](7)))).
			To(BeFalse())
		Expect(b.Close()).To(Succeed())
		Expect(a.Close()).To(Succeed())
		release := MustSucceed(db.svc.ClaimDelete(keys))
		release()
	})
})
