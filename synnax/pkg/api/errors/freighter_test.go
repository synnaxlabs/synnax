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
	roacherrors "github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter/ferrors"
	"github.com/synnaxlabs/synnax/pkg/api/errors"
)

var _ = Describe("Freighter", func() {
	Describe("Encode + Decode", func() {
		Context("validation Err", func() {
			It("Should encode and decode a validation error", func() {
				err := errors.Validation(errors.Fields{
					{
						Field:   "field",
						Message: "message",
					},
					{
						Field:   "field2",
						Message: "message2",
					},
				})
				encoded := ferrors.Encode(err)
				decoded := ferrors.Decode(encoded)
				Expect(decoded).To(Equal(err))
			})
		})
		Context("Message Err", func() {
			It("Should encode and decode a message error", func() {
				err := errors.General(roacherrors.New("my crazy error"))
				encoded := ferrors.Encode(err)
				decoded := ferrors.Decode(encoded)
				Expect(decoded).To(Equal(err))
			})
		})
		Context("Nil Err", func() {
			It("Should encode and decode a nil error", func() {
				err := errors.Nil
				encoded := ferrors.Encode(err)
				decoded := ferrors.Decode(encoded)
				Expect(decoded).To(BeNil())
			})
		})
	})

})
