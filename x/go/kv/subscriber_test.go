// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/observe"
)

type dataStruct struct {
	Value []byte
}

var _ = Describe("Flush", func() {
	It("Should flush the observable contents", func() {
		o := observe.New[dataStruct]()
		db := memkv.New()
		ecd := &binary.GobCodec{}
		flush := &kv.Subscriber[dataStruct]{
			Key:         []byte("key"),
			Store:       db,
			MinInterval: 5 * time.Millisecond,
			Encoder:     ecd,
		}
		o.OnChange(flush.Flush)

		o.Notify(ctx, dataStruct{Value: []byte("hello")})
		o.Notify(ctx, dataStruct{Value: []byte("world")})

		Eventually(func(g Gomega) {
			b, closer, err := db.Get(ctx, []byte("key"))
			g.Expect(err).ToNot(HaveOccurred())
			var ds dataStruct
			g.Expect(ecd.Decode(ctx, b, &ds)).To(Succeed())
			g.Expect(ds.Value).To(Equal([]byte("hello")))
			Expect(closer.Close()).To(Succeed())
		}).Should(Succeed())
	})
})
