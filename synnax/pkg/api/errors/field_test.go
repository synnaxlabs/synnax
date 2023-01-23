// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package errors_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/errors"
)

var _ = Describe("Field", func() {
	Describe("Err", func() {
		It("Should return a string representation of the Field error", func() {
			Expect(errors.Field{Field: "field", Message: "Message"}.Error()).To(Equal("field: Message"))
		})
	})
	Describe("Fields", func() {
		Describe("Err", func() {
			It("Should return a string representation of the Field error", func() {
				Expect(errors.Fields{
					{Field: "field", Message: "Message"},
					{Field: "field2", Message: "message2"},
				}.Error()).To(Equal("field: Message\nfield2: message2"))
			})
		})
	})
})
