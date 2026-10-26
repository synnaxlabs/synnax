// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apigroup "github.com/synnaxlabs/synnax/pkg/api/group"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Retrieve", func() {
	It("Should reject the request when retrieve is not granted on a match", func(
		ctx SpecContext,
	) {
		g := MustSucceed(
			svc.Group.NewWriter(nil).Create(ctx, "unretrievable", ontology.RootID),
		)
		Expect(apiSvc.Retrieve(AuthedCtx(ctx, author), apigroup.RetrieveRequest{
			Where: &group.Where{Key: &filter.Set[group.Key]{Eq: &g.Key}},
		})).Error().To(MatchError(access.ErrDenied))
	})
})
