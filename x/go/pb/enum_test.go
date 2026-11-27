// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/pb"
	"github.com/synnaxlabs/x/validate"
)

type port string

func (p port) IsValid() bool { return p == "DIO0" || p == "DIO1" }

var _ = Describe("Enum", func() {
	It("Should convert a member of the enum", func() {
		Expect(pb.Enum[port]("DIO1")).To(Equal(port("DIO1")))
	})

	It("Should reject a value that no member has", func() {
		Expect(pb.Enum[port]("DIO9")).Error().To(MatchError(validate.ErrValidation))
	})
})
