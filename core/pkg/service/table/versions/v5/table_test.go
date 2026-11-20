// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	v5 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v5"
)

var _ = Describe("Table", func() {
	Describe("GorpKey", func() {
		It("Should return the table's key", func() {
			k := uuid.New()
			Expect(v5.Table{Key: k}.GorpKey()).To(Equal(k))
		})
	})

	Describe("ResourceID", func() {
		It("Should return the table resource identifier", func() {
			k := uuid.New()
			Expect(v5.Table{Key: k}.ResourceID()).To(Equal(resource.ID{
				Type: resource.TypeTable, Key: k.String(),
			}))
		})
	})
})
