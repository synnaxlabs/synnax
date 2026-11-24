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

var _ = Describe("ReferencedTables", func() {
	It("Should name each table that a reference of an open table points at", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[int32, benchEntry]{
			DB:   db,
			Name: "ref_target",
		}))
		Expect(db.ReferencedTables()).To(BeEmpty())
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[int32, refBenchHolder]{
			DB:         db,
			Name:       "ref_holder",
			References: []gorp.Reference[int32, refBenchHolder]{newRefBenchReference()},
		}))
		Expect(db.ReferencedTables()).To(Equal([]string{"ref_target"}))
	})
})
