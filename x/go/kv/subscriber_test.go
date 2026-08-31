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
	"github.com/synnaxlabs/x/encoding/gob"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/observe"
	. "github.com/synnaxlabs/x/testutil"
)

type dataStruct struct {
	Value []byte
}

var _ = Describe("Flush", func() {
	It("Should flush the observable contents", func(ctx SpecContext) {
		o := observe.New[dataStruct]()
		db := DeferClose(memkv.New())
		codec := gob.Codec
		// The interval only has to outlast the two Notify calls below. A short one
		// races the scheduler: once it expires between them, the second write lands
		// and no later read can recover the first.
		flush := &kv.Subscriber{
			Key:         []byte("key"),
			Store:       db,
			MinInterval: time.Hour,
			Encoder:     codec,
		}
		o.OnChange(flush.Flush)

		o.Notify(ctx, dataStruct{Value: []byte("hello")})
		o.Notify(ctx, dataStruct{Value: []byte("world")})

		// Flush writes on the calling goroutine, so both writes are already done.
		b, closer := MustSucceed2(db.Get(ctx, []byte("key")))
		var ds dataStruct
		Expect(codec.Decode(ctx, b, &ds)).To(Succeed())
		Expect(ds.Value).To(Equal([]byte("hello")))
		Expect(closer.Close()).To(Succeed())
	})

	It("Should write the state before returning to the caller", func(ctx SpecContext) {
		db := DeferClose(memkv.New())
		codec := gob.Codec
		flush := &kv.Subscriber{
			Key:     []byte("key"),
			Store:   db,
			Encoder: codec,
		}

		flush.Flush(ctx, dataStruct{Value: []byte("hello")})

		b, closer := MustSucceed2(db.Get(ctx, []byte("key")))
		var ds dataStruct
		Expect(codec.Decode(ctx, b, &ds)).To(Succeed())
		Expect(ds.Value).To(Equal([]byte("hello")))
		Expect(closer.Close()).To(Succeed())
	})
})
