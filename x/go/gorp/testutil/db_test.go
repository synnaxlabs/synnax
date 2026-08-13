// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/gorp"
	. "github.com/synnaxlabs/x/gorp/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

type entry struct {
	ID   int32  `msgpack:"id"`
	Data string `msgpack:"data"`
}

func (e entry) GorpKey() int32    { return e.ID }
func (e entry) SetOptions() []any { return nil }

var _ = Describe("OpenGorpMsgpackDB", func() {
	It(
		"Should store entries that do not implement orc.SelfCodec",
		func(ctx SpecContext) {
			db := OpenGorpMsgpackDB()
			defer func() { Expect(db.Close()).To(Succeed()) }()
			w := gorp.WrapWriter[int32, entry](db)
			Expect(w.Set(ctx, entry{ID: 1, Data: "one"})).To(Succeed())
			r := gorp.WrapReader[int32, entry](db)
			Expect(MustSucceed(r.Get(ctx, 1))).To(Equal(entry{ID: 1, Data: "one"}))
		},
	)
})
