// Copyright 2022 Synnax Labs, Inc.
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
	kvx "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
)

var _ = Describe("IteratorServer", func() {
	var (
		kv   kvx.DB
		ecdc binary.EncoderDecoder
	)
	BeforeEach(func() {
		kv = memkv.New()
		ecdc = &binary.GobEncoderDecoder{}
		val, err := ecdc.Encode(map[string]string{"key1": "value1", "key2": "value2"})
		Expect(err).To(BeNil())
		Expect(kv.Set([]byte("key1"), val)).To(Succeed())
		Expect(kv.Set([]byte("key2"), val)).To(Succeed())

	})
	AfterEach(func() {
		Expect(kv.Close()).To(Succeed())
	})
	It("Should decode values before returning them to the caller", func() {
		iter := gorp.WrapKVIter[map[string]string](
			kv.NewIterator(kvx.PrefixIter([]byte("key"))),
			gorp.WithEncoderDecoder(ecdc),
		)
		for iter.First(); iter.Valid(); iter.Next() {
			Expect(iter.Value()).To(Equal(map[string]string{"key1": "value1", "key2": "value2"}))
		}
		Expect(iter.Error()).To(BeNil())
		Expect(iter.Close()).To(Succeed())
	})
})
