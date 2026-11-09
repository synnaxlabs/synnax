// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package video_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/video"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resources", func() {
	It("Should name each view by its name", func(ctx SpecContext) {
		v := create(ctx, video.Video{Name: "resource-video"})
		Expect(svc.Resources().Retrieve(ctx, db, []resource.ID{v.ResourceID()})).
			To(Equal([]resource.Resource{{ID: v.ResourceID(), Name: v.Name}}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(svc.Resources().Retrieve(
			ctx,
			db,
			[]resource.ID{{Type: resource.TypeVideo, Key: "pump"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid video key "pump"`)),
		))
	})

	It("Should find a view by its name", func(ctx SpecContext) {
		v := create(ctx, video.Video{Name: "turbopump camera"})
		q := search.Query[resource.ID]{Term: "turbopump camera"}
		Expect(MustSucceed(svc.Resources().Search(ctx, db, q)).Keys).
			To(ContainElement(video.ResourceID(v.Key)))
	})
})
