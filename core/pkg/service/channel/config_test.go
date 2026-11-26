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
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("ServiceConfig", func() {
	It(
		"Should fail to open a service with no time-series database",
		func(ctx SpecContext) {
			node := testutil.NewNode(ctx)
			cfg, _ := serviceConfig(ctx, node, node.DB)
			cfg.TS = nil
			Expect(channel.OpenService(ctx, cfg)).Error().To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("ts: must be non-nil")),
			))
		},
	)
})
