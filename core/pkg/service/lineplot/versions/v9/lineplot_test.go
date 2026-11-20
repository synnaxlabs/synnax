// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v9_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v9 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v9"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

var _ = Describe("LinePlot", func() {
	Describe("GorpKey", func() {
		It("Should return the line plot's key", func() {
			k := uuid.New()
			Expect(v9.LinePlot{Key: k}.GorpKey()).To(Equal(k))
		})
	})

	Describe("ResourceID", func() {
		It("Should return the line plot resource identifier", func() {
			k := uuid.New()
			Expect(v9.LinePlot{Key: k}.ResourceID()).To(Equal(resource.ID{
				Type: resource.TypeLineplot, Key: k.String(),
			}))
		})
	})
})
