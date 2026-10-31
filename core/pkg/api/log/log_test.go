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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apilog "github.com/synnaxlabs/synnax/pkg/api/log"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/x/filter"
)

var _ = Describe("Retrieve", func() {
	It("Should reject the request when retrieve is not granted on a match", func(
		ctx SpecContext,
	) {
		r := log.Log{Name: "unretrievable"}
		Expect(svc.Log.NewWriter(nil).Create(ctx, uuid.Nil(), &r)).To(Succeed())
		Expect(apiSvc.Retrieve(AuthedCtx(ctx, author), apilog.RetrieveRequest{
			Where: &log.Where{Key: &filter.Set[log.Key]{Eq: &r.Key}},
		})).Error().To(MatchError(access.ErrDenied))
	})
})
