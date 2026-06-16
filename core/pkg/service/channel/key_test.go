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
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("ParseKey", func() {
	It("Should correctly parse a key from its string representation", func() {
		Expect(MustSucceed(channel.ParseKey("123456"))).To(Equal(channel.Key(123456)))
	})
	It("Should return an error when the key is not a valid integer", func() {
		Expect(channel.ParseKey("123456a")).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("123456a is not a valid channel key")),
		))
	})
})
