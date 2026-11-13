// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	v4 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v4"
)

var _ = Describe("Task", func() {
	Describe("GorpKey", func() {
		It("Should return the task's key", func() {
			k := uuid.New()
			Expect(v4.Task{Key: k}.GorpKey()).To(Equal(k))
		})
	})

	Describe("ResourceID", func() {
		It("Should return the task resource identifier", func() {
			k := uuid.New()
			Expect(v4.Task{Key: k}.ResourceID()).To(Equal(resource.ID{
				Type: resource.TypeTask, Key: k.String(),
			}))
		})
	})

	Describe("String", func() {
		It("Should return the name and key of a named task", func() {
			k := uuid.New()
			Expect(v4.Task{Key: k, Name: "pump"}.String()).
				To(Equal("[pump]<" + k.String() + ">"))
		})

		It("Should return the key of an unnamed task", func() {
			k := uuid.New()
			Expect(v4.Task{Key: k}.String()).To(Equal(k.String()))
		})
	})
})
