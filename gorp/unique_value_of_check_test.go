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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("UniqueIndex ValueOf check", func() {
	It("Should fail on a target in a table that no index of the space holds", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, uniqueFolder]{
			DB:      db,
			Name:    "folder",
			Indexes: []gorp.Index[string, uniqueFolder]{newFolderIndex()},
		}))
		itemIndex := newItemIndex()
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, uniqueItem]{
			DB:         db,
			Name:       "item",
			Indexes:    []gorp.Index[uint32, uniqueItem]{itemIndex},
			References: []gorp.Reference[uint32, uniqueItem]{newItemFolderRef()},
		}))
		Expect(itemIndex.ValueOf(ctx, db, gorp.NewTarget("other", "d"))).Error().
			To(MatchError(ContainSubstring("other")))
	})
})
