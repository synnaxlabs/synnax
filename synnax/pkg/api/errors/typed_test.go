// Copyright 2022 Synnax Labs, Inc.
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

var _ = Describe("Typed", func() {
	Describe("Err", func() {
		It("Should return a string representation of the error", func() {
			err := errors.Typed{Type: "type", Err: errors.Field{Field: "field", Message: "Message"}}
			Expect(err.Error()).To(Equal("field: Message"))
		})
		It("Should return 'nil' when the error is of type Nil", func() {
			err := errors.Nil
			Expect(err.Error()).To(Equal("nil"))
		})
	})
	Describe("Occurred", func() {
		It("Should return true when the error is not of type Nil", func() {
			err := errors.Typed{Type: "type", Err: errors.Field{Field: "field", Message: "Message"}}
			Expect(err.Occurred()).To(BeTrue())
		})
	})
})
