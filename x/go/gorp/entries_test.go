// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
)

var _ = Describe("Entries", func() {
	Describe("Get and Set", func() {
		It("Should return an empty slice if no entries were set on the query", func() {
			q := gorp.NewRetrieve[int, entry]()
			entries := gorp.GetEntries[int, entry](q)
			Expect(entries.All()).To(HaveLen(0))
		})
		It("Should panic if a caller attempts to set multiple entries on a single entry query", func() {
			q := query.New()
			gorp.SetEntry[int, entry](q, &entry{})
			e := gorp.GetEntries[int, entry](q)
			Expect(func() {
				e.Set(2, entry{})
			}).To(Panic())
		})
	})
	Describe("TypePrefix", func() {
		It("Should not append a type prefix to a particular key when type prefix is off", func() {
			db := memkv.New()
			gorpDB := gorp.Wrap(db,
				gorp.WithoutTypePrefix(),
			)
			Expect(gorp.NewCreate[int, entry]().
				Entries(&[]entry{{ID: 1, Data: "data"}}).
				Exec(gorpDB.BeginWrite(ctx))).To(Succeed())
			// use msgpack to encode the entry int 1  into a byte slice
			ecd := &binary.MsgPackEncoderDecoder{}
			b, err := ecd.Encode(1)
			Expect(err).To(Not(HaveOccurred()))
			_, err = db.NewReader(ctx).Get(b)
			Expect(err).To(Not(HaveOccurred()))
		})
	})

})
