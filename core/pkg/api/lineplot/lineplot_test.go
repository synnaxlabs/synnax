// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apilineplot "github.com/synnaxlabs/synnax/pkg/api/lineplot"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/x/filter"
)

var _ = Describe("Retrieve", func() {
	It("Should reject the request when retrieve is not granted on a match", func(
		ctx SpecContext,
	) {
		r := lineplot.LinePlot{Name: "unretrievable"}
		Expect(svc.LinePlot.NewWriter(nil).Create(ctx, uuid.Nil(), &r)).To(Succeed())
		Expect(apiSvc.Retrieve(AuthedCtx(ctx, author), apilineplot.RetrieveRequest{
			Where: &lineplot.Where{Key: &filter.Set[lineplot.Key]{Eq: &r.Key}},
		})).Error().To(MatchError(access.ErrDenied))
	})
})
