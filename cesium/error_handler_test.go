// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	"os"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/place"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap/zapcore"
)

var _ = Describe("Error handler", func() {
	var (
		mu      sync.Mutex
		handled []error
	)
	handle := func(err error) {
		mu.Lock()
		defer mu.Unlock()
		handled = append(handled, err)
	}
	errs := func() []error {
		mu.Lock()
		defer mu.Unlock()
		return handled
	}
	BeforeEach(func() { handled = nil })

	It("Should get a directory of the root that names no channel", func(
		ctx SpecContext,
	) {
		fs := xfs.NewMem()
		MustSucceed(fs.Sub("not-a-key"))
		mustOpenDBOnFS(ctx, fs, cesium.WithErrorHandler(handle))
		Expect(errs()).To(ConsistOf(
			MatchError(ContainSubstring("failed to parse the directory not-a-key")),
		))
	})

	It("Should get the corrupt batches of delete intents that open skips", func(
		ctx SpecContext,
	) {
		fs := xfs.NewMem()
		db := MustSucceed(cesium.Open(ctx, "", cesium.WithFS(fs)))
		// A damaged batch is corrupt only before an intact one; a damaged last batch is
		// a torn tail.
		for range 2 {
			key := GenerateChannelKey()
			Expect(db.CreateChannel(ctx, cesium.Channel{
				Key: key, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			})).To(Succeed())
			Expect(db.Intend([]cesium.ChannelKey{key})).To(Succeed())
		}
		Expect(db.Close()).To(Succeed())
		f := MustSucceed(fs.Open("delete-intents", os.O_RDWR))
		b := make([]byte, 1)
		MustSucceed(f.ReadAt(b, 4))
		b[0] ^= 0xFF
		MustSucceed(f.WriteAt(b, 4))
		Expect(f.Close()).To(Succeed())
		mustOpenDBOnFS(ctx, fs, cesium.WithErrorHandler(handle))
		Expect(errs()).To(ConsistOf(MatchError(place.ErrCorrupt)))
	})

	It("Should log each error at the Error level when no handler is set", func(
		ctx SpecContext,
	) {
		fs := xfs.NewMem()
		MustSucceed(fs.Sub("not-a-key"))
		ins, logs := ObservedInstrumentation(zapcore.ErrorLevel)
		mustOpenDBOnFS(ctx, fs, cesium.WithInstrumentation(ins))
		survived := logs.FilterMessage("survived a storage fault").All()
		Expect(survived).To(HaveLen(1))
		Expect(survived[0].ContextMap()["error"]).
			To(ContainSubstring("failed to parse the directory not-a-key"))
	})
})
