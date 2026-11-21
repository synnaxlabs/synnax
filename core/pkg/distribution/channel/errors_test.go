// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Error", func() {
	It("Should name the key of its channel and keep the error it wraps", func() {
		err := channel.Error{Key: 7, Err: errors.Wrap(validate.ErrValidation, "bad")}
		Expect(err.Error()).To(Equal("channel <7>: bad: validation error"))
		Expect(err).To(MatchError(validate.ErrValidation))
	})
})
