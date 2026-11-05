// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apilog "github.com/synnaxlabs/synnax/pkg/api/log"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Retrieve", func() {
	It("Should return no logs for a request with no filter", func(ctx SpecContext) {
		u := newCreator(ctx)
		MustSucceed(apiSvc.Create(AuthedCtx(ctx, u), nil, apilog.CreateRequest{
			Project: proj.Key,
			Logs:    []log.Log{{Name: "unfiltered"}},
		}))
		Expect(apiSvc.Retrieve(AuthedCtx(ctx, u), apilog.RetrieveRequest{})).
			To(Equal(apilog.RetrieveResponse{Logs: []log.Log{}}))
	})
})
