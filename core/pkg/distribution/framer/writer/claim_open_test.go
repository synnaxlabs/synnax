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
	"fmt"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Concurrent opens on the pointer log", func() {
	It(
		"Should open every session on one data key at once with one live claim",
		func(ctx SpecContext) {
			host := newNode(ctx, true)
			idx := MustSucceed(host.CreateChannels(ctx, []channel.Channel{{
				Name:     "concurrent_time",
				IsIndex:  true,
				DataType: telem.TimestampT,
			}}))[0]
			data := MustSucceed(host.CreateChannels(ctx, []channel.Channel{{
				Name:       "concurrent",
				DataType:   telem.Int64T,
				LocalIndex: idx.LocalKey,
			}}))[0]
			// A pick and a claim that did not run under one lock would let two
			// sessions open Cesium writers on the key, and the pointer log refuses
			// the second.
			const sessions = 8
			var (
				wg      sync.WaitGroup
				writers [sessions]*writer.Writer
				errs    [sessions]error
			)
			for i := range sessions {
				wg.Go(func() {
					writers[i], errs[i] = host.Framer.OpenWriter(ctx, writer.Config{
						ControlSubject: control.Subject{Key: fmt.Sprintf("open-%d", i)},
						Keys:           channel.Keys{data.Key()},
						Start:          10 * telem.SecondTS,
						Authorities: []control.Authority{
							control.AuthorityAbsolute - control.Authority(i),
						},
					})
				})
			}
			wg.Wait()
			for i := range sessions {
				Expect(errs[i]).ToNot(HaveOccurred())
			}
			for _, w := range writers {
				Expect(w.Close()).To(Succeed())
			}
		},
	)
})
