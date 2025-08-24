// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package uuid_test

import (
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xuuid "github.com/synnaxlabs/x/uuid"
)

var _ = Describe("UUID", func() {
	Describe("EncodeStringBytes", func() {
		Specify("should encode a UUID to a byte slice", func() {
			uuid := uuid.New()
			dst := make([]byte, 36)
			xuuid.EncodeStringBytes(dst, uuid)
			Expect(string(dst)).To(Equal(uuid.String()))
		})
		Specify("should panic if the byte slice is not long enough", func() {
			uuid := uuid.New()
			dst := make([]byte, 35)
			Expect(func() { xuuid.EncodeStringBytes(dst, uuid) }).To(Panic())
		})
		Specify("should not make any allocations", func() {
			uuid := uuid.New()
			dst := make([]byte, 36)
			allocs := testing.AllocsPerRun(1000, func() {
				xuuid.EncodeStringBytes(dst, uuid)
			})
			Expect(allocs).To(BeZero())
		})
	})
})
