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
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Errors", func() {
	Describe("NewNotFoundError", func() {
		It("Should return an error with the correct message", func() {
			err := channel.NewNotFoundError(1)
			Expect(err).To(HaveOccurredAs(channel.ErrNotFound))
			Expect(err).To(MatchError(ContainSubstring("channel with key 1 not found")))
		})
	})

	Describe("NewErrorWrapper", func() {
		It("Should return an error with the correct message", func() {
			ch := channel.Channel{Key: 1, Name: "foo"}
			err := channel.NewErrorWrapper(ch)(errors.Newf("bad error"))
			Expect(err).To(MatchError(ContainSubstring("channel [foo]<1>: bad error")))
		})
		It("Should return nil if the error is nil", func() {
			ch := channel.Channel{Key: 1, Name: "foo"}
			err := channel.NewErrorWrapper(ch)(nil)
			Expect(err).To(BeNil())
		})
	})
})
