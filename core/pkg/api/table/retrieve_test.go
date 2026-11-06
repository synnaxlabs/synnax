// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apitable "github.com/synnaxlabs/synnax/pkg/api/table"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Retrieve", func() {
	It("Should return no tables for a request with no filter", func(ctx SpecContext) {
		u := newCreator(ctx)
		MustSucceed(apiSvc.Create(AuthedCtx(ctx, u), nil, apitable.CreateRequest{
			Tables: []table.Table{{Project: &proj.Key, Name: "unfiltered"}},
		}))
		Expect(apiSvc.Retrieve(AuthedCtx(ctx, u), apitable.RetrieveRequest{})).
			To(Equal(apitable.RetrieveResponse{Tables: []table.Table{}}))
	})
})
