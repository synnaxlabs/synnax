// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	"encoding/binary"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("FilterReferenced checks", func() {
	var (
		db       *gorp.DB
		channels *gorp.Table[uint32, usedChannel]
		used     = func(referenced bool) gorp.Filter[uint32, usedChannel] {
			return channels.FilterReferenced(referenced, usedReadTag, usedWriteTag)
		}
		keysOf = func(ctx SpecContext, f gorp.Filter[uint32, usedChannel]) []uint32 {
			GinkgoHelper()
			var found []usedChannel
			Expect(channels.NewRetrieve().Where(f).Entries(&found).Exec(ctx, db)).
				To(Succeed())
			keys := make([]uint32, len(found))
			for i, c := range found {
				keys[i] = c.Key
			}
			return keys
		}
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		channels = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, usedChannel]{
			DB:     db,
			Name:   "channel",
			Issued: func(uint32) bool { return true },
		}))
		symbols := openUsedHolder(
			ctx, db, "symbol", usedReadTag, gorp.Weak,
			func(e *usedSymbol) *[]uint32 { return &e.Channels },
		)
		for k := uint32(1); k <= 3; k++ {
			Expect(channels.NewCreate().Entry(&usedChannel{Key: k}).Exec(ctx, db)).
				To(Succeed())
		}
		Expect(symbols.NewCreate().
			Entry(&usedSymbol{Key: "a", Channels: []uint32{1}}).
			Exec(ctx, db)).To(Succeed())
		corrupt[uint32, usedChannel](ctx, db, 1)
	})

	It("Should never decode a channel that a Not of the filter leaves out", func(
		ctx SpecContext,
	) {
		Expect(keysOf(ctx, gorp.Not(used(true)))).To(ConsistOf(uint32(2), uint32(3)))
	})

	It("Should never decode a channel that an Or of the filter leaves out", func(
		ctx SpecContext,
	) {
		// Or decides before decode only when each child has only a raw check.
		third := gorp.MatchRaw[uint32, usedChannel](func(key, _ []byte) (bool, error) {
			return binary.BigEndian.Uint32(key[len(key)-4:]) == 3, nil
		})
		Expect(keysOf(ctx, gorp.Or(used(false), third))).
			To(ConsistOf(uint32(2), uint32(3)))
	})
})
