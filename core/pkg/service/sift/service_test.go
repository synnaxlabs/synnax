// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sift_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/sift"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Service", func() {
	It("Should require a DB", func(ctx SpecContext) {
		Expect(sift.OpenService(ctx, sift.ServiceConfig{})).Error().
			To(MatchError(ContainSubstring("db: must be non-nil")))
	})

	It("Should own the scan task config store", func(ctx SpecContext) {
		svcDB := DeferClose(gorp.Wrap(memkv.New()))
		svc := MustOpen(sift.OpenService(ctx, sift.ServiceConfig{DB: svcDB}))
		Expect(svc.Stores()).To(HaveExactElements(
			WithTransform(config.Store.Type, Equal(sift.ScanTaskType)),
		))
	})
})
