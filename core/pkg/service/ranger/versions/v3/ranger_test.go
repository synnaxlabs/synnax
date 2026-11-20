// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v3 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

var _ = Describe("Range", func() {
	Describe("GorpKey", func() {
		It("Should return the range's key", func() {
			k := uuid.New()
			Expect(v3.Range{Key: k}.GorpKey()).To(Equal(k))
		})
	})

	Describe("ResourceID", func() {
		It("Should return the range resource identifier", func() {
			k := uuid.New()
			Expect(v3.Range{Key: k}.ResourceID()).To(Equal(resource.ID{
				Type: resource.TypeRange, Key: k.String(),
			}))
		})
	})
})
