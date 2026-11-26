// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package impact_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	apiimpact "github.com/synnaxlabs/synnax/pkg/api/impact"
	"github.com/synnaxlabs/synnax/pkg/service/impact"
	. "github.com/synnaxlabs/x/testutil"
)

// planOf returns a handler that answers as h does, with the plan of the delete that
// the response of h carries.
func planOf[RQ freighter.Payload](
	h freighter.UnaryHandler[RQ, apiimpact.Response],
) freighter.UnaryHandler[RQ, impact.Plan] {
	return func(ctx context.Context, req RQ) (impact.Plan, error) {
		res, err := h(ctx, req)
		if err != nil {
			return impact.Plan{}, err
		}
		return res.Variant.(apiimpact.DeleteResponse).Plan, nil
	}
}

var _ = ShouldNotLeakGoroutinesPerSpec()

func TestImpact(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Impact Suite", SuiteConfig())
}
