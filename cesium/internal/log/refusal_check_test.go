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
	"encoding/binary"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Log refusal check", func() {
	It("Should take a Commit after an entry with no Commit that does not fit", func() {
		fs := xfs.NewMem()
		Expect(log.Prepare(fs, generation, logSize)).To(Succeed())
		l := MustSucceed(log.Start(log.Config{
			FS:         fs,
			Executor:   goExec,
			DB:         db,
			Generation: generation,
			Sequence:   5,
		}))
		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)
		ran := make(chan error, 1)
		go func() { ran <- l.Run(ctx) }()
		Expect(l.Append(bigEntry(logSize)).Wait()).Error().To(MatchError(
			fmt.Sprintf("an entry of %d bytes %s", logSize, full),
		))
		Expect(l.Append(commitEntry(64)).Wait()).To(Equal(uint64(5)))
		Expect(l.Close()).To(Succeed())
		Eventually(ran).Should(Receive(BeNil()))
	})

	It("Should read a header copy that fails its check as corrupt, also when its "+
		"version is unknown", func() {
		b := header.Encode(0)
		binary.LittleEndian.PutUint16(b[8:], 2)
		_, err := log.DecodeHeader(b, db, generation, 0)
		Expect(err).To(MatchError(place.ErrCorrupt))
		Expect(err).To(MatchError(ContainSubstring(
			"log header at 0 fails its check for generation 3",
		)))
		Expect(err).ToNot(MatchError(log.ErrUnknown))
	})
})
