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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/kv"
)

var _ = Describe("KVKey", func() {
	Describe("CompositeKey", func() {
		It("Should generate a composite key from elements", func() {
			key, err := kv.CompositeKey("foo", int8(1))
			Expect(err).ToNot(HaveOccurred())
			Expect(key).To(Equal([]byte{102, 111, 111, 1}))
		})
	})
})
