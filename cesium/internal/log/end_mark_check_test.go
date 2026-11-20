// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("End mark check", func() {
	It("Should write the clean mark inside the file after a mark in its last block",
		func() {
			fs := newOpFS()
			Expect(log.Prepare(fs, generation, logSize)).To(Succeed())
			l := MustSucceed(log.Start(log.Config{
				FS:         fs,
				DB:         db,
				Generation: generation,
				Sequence:   5,
				Size:       logSize,
			}))
			ctx, cancel := context.WithCancel(context.Background())
			DeferCleanup(cancel)
			ran := make(chan error, 1)
			go func() { ran <- l.Run(ctx) }()
			// The frame of this delete ends 100 bytes before the last block, so the
			// idle mark after it starts the last block.
			entry := make(
				[]byte,
				logSize-log.BlockSize-100-log.FramesStart-log.FrameHeaderSize,
			)
			Expect(l.Append(entry, true).Wait()).To(Equal(uint64(5)))
			Eventually(func() int { return len(walk(fileBytes(fs))) }).Should(Equal(2))
			Expect(l.Close()).To(Succeed())
			Expect(<-ran).To(Succeed())
			b := fileBytes(fs)
			Expect(len(b)).To(Equal(logSize), "the log file grew past its size")
			frames := walk(b)
			Expect(frames[len(frames)-1].Clean).To(BeTrue())
		},
	)
})
