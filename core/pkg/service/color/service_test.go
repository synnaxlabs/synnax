// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package color_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("OpenService", func() {
	It("Should refuse a config with no DB", func(ctx SpecContext) {
		Expect(color.OpenService(ctx, color.ServiceConfig{})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("db: must be non-nil")),
		))
	})
})
