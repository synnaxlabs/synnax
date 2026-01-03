// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package core_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"

	"github.com/synnaxlabs/cesium/internal/core"
)

var _ = Describe("Errors", func() {
	Describe("NewErrChannelNotFound", func() {
		It("Should return an error with the correct message", func() {
			err := core.NewErrChannelNotFound(1)
			Expect(err).To(HaveOccurredAs(core.ErrChannelNotFound))
			Expect(err).To(MatchError(ContainSubstring("channel with key 1 not found")))
		})
	})

	Describe("NewErrResourceClosed", func() {
		It("Should return an error with the correct message", func() {
			err := core.NewErrResourceClosed("writer")
			Expect(err).To(HaveOccurredAs(core.ErrClosedResource))
			Expect(err).To(MatchError(ContainSubstring("cannot complete operation on closed writer")))
		})
	})

	Describe("NewChannelErrWrapper", func() {
		It("Should return an error with the correct message", func() {
			ch := core.Channel{Key: 1, Name: "foo"}
			err := core.NewChannelErrWrapper(ch)(errors.Newf("bad error"))
			Expect(err).To(MatchError(ContainSubstring("channel [foo]<1>: bad error")))
		})
		It("Should return nil if the error is nil", func() {
			ch := core.Channel{Key: 1, Name: "foo"}
			err := core.NewChannelErrWrapper(ch)(nil)
			Expect(err).To(BeNil())
		})
	})
})
