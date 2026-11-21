// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

// deleteCases counts the crash images that a recovery finds in each state of a delete.
type deleteCases struct {
	// failedCommit is an intent whose record survived.
	failedCommit int
	// committed is an intent whose record is gone.
	committed int
	// cutRemoval is a removal that a crash cut short.
	cutRemoval int
	// cutClear is an intent whose directory is gone: a crash came before its clear.
	cutClear int
	// tornTail is an intents file that ends in a batch that a crash tore.
	tornTail int
}

// files counts the states that the Cesium files of image show, and returns the keys
// with an intent. Each batch of the intents file is a count, a CRC32C of the batch's
// offset and records, then the records. A record is a key and flags, and a record with
// the clear flag clears its key.
func (c *deleteCases) files(image *fault.FS) []cesium.ChannelKey {
	GinkgoHelper()
	if !MustSucceed(image.Exists("cesium")) {
		return nil
	}
	for _, info := range MustSucceed(image.List("cesium")) {
		if strings.HasSuffix(info.Name(), "-DELETE") {
			c.cutRemoval++
		}
	}
	if !MustSucceed(image.Exists("cesium/delete-intents")) {
		return nil
	}
	f := MustSucceed(image.Open("cesium/delete-intents", os.O_RDONLY))
	defer func() { Expect(f.Close()).To(Succeed()) }()
	data := make([]byte, MustSucceed(f.Stat()).Size())
	if len(data) > 0 {
		MustSucceed(f.ReadAt(data, 0))
	}
	const headerSize, recordSize, clearFlag = 8, 8, 1
	var (
		keys  = set.New[cesium.ChannelKey]()
		table = crc32.MakeTable(crc32.Castagnoli)
		good  int
	)
	for off := 0; off+headerSize <= len(data); {
		count := int(binary.LittleEndian.Uint32(data[off:]))
		end := off + headerSize + count*recordSize
		if count == 0 || end > len(data) {
			break
		}
		records := data[off+headerSize : end]
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], uint64(off))
		sum := crc32.Update(crc32.Checksum(b[:], table), table, records)
		if binary.LittleEndian.Uint32(data[off+4:]) == sum {
			good = end
			for at := 0; at < len(records); at += recordSize {
				key := binary.LittleEndian.Uint32(records[at:])
				if binary.LittleEndian.Uint32(records[at+4:])&clearFlag != 0 {
					keys.Remove(key)
				} else {
					keys.Add(key)
				}
			}
		}
		off = end
	}
	if good < len(data) {
		c.tornTail++
	}
	for k := range keys {
		if !MustSucceed(image.Exists(fmt.Sprintf("cesium/%d", k))) {
			c.cutClear++
		}
	}
	return keys.Slice()
}

// records counts the intents of keys whose records survived in e, and those whose
// records are gone.
func (c *deleteCases) records(
	ctx context.Context,
	e *crashEngine,
	keys []cesium.ChannelKey,
) error {
	for _, k := range keys {
		var ch channel.Channel
		err := e.svc.NewRetrieve().
			Where(channel.MatchKeys(channel.Key(k))).
			Entry(&ch).
			Exec(ctx, e.db)
		switch {
		case err == nil:
			c.failedCommit++
		case errors.Is(err, query.ErrNotFound):
			c.committed++
		default:
			return err
		}
	}
	return nil
}

// run runs the crash table in one mode under profile, and counts the cases it reaches.
func (c *deleteCases) run(
	ctx context.Context,
	profile crash.Profile,
	dirsDurable bool,
) {
	GinkgoHelper()
	_, err := crash.Run(ctx, crash.Config{
		Open: func(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
			image, ok := fs.(*fault.FS)
			var keys []cesium.ChannelKey
			if ok {
				keys = c.files(image)
			}
			e, err := openEngine(ctx, fs, dirsDurable)
			if err != nil || !ok {
				return e, err
			}
			return e, c.records(ctx, e, keys)
		},
		Commits:    4,
		Stride:     1,
		Profile:    profile,
		Exhaustive: true,
	})
	var v *crash.Violation
	Expect(err == nil || errors.As(err, &v)).To(BeTrue(), "run failed: %v", err)
}

var _ = Describe("Crash adapter, delete cases", func() {
	It("Should reach each crash case of a delete in the crash table",
		func(ctx SpecContext) {
			var cases deleteCases
			for _, profile := range []crash.Profile{crash.Strict, crash.Partial} {
				cases.run(ctx, profile, false)
				cases.run(ctx, profile, true)
			}
			Expect(cases.failedCommit).To(BeNumerically(">", 0))
			Expect(cases.committed).To(BeNumerically(">", 0))
			Expect(cases.cutRemoval).To(BeNumerically(">", 0))
			Expect(cases.cutClear).To(BeNumerically(">", 0))
			Expect(cases.tornTail).To(BeNumerically(">", 0))
		},
	)
})
