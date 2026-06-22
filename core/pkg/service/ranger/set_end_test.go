// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("SetEnd", func() {
	// SetEnd opens its own tx, so ranges must be committed (nil-tx) and cleaned up.
	create := func(ctx SpecContext, tr telem.TimeRange) ranger.Range {
		r := ranger.Range{Name: "set_end_" + uuid.NewString(), TimeRange: tr}
		Expect(svc.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(svc.NewWriter(nil).Delete(ctx, r.Key)).To(Succeed())
		})
		return r
	}
	retrieve := func(ctx SpecContext, key ranger.Key) ranger.Range {
		var r ranger.Range
		Expect(svc.NewRetrieve().Where(ranger.MatchKeys(key)).Entry(&r).Exec(ctx, nil)).To(Succeed())
		return r
	}

	It("Should set the end bound while preserving the start", func(ctx SpecContext) {
		r := create(ctx, telem.TimeRange{Start: telem.SecondTS, End: telem.TimeStampMax})
		end := telem.SecondTS * 10
		Expect(svc.SetEnd(ctx, r.Key, end)).To(Succeed())

		updated := retrieve(ctx, r.Key)
		Expect(updated.TimeRange.Start).To(Equal(telem.SecondTS))
		Expect(updated.TimeRange.End).To(Equal(end))
	})

	It("Should overwrite a previously set end bound", func(ctx SpecContext) {
		r := create(ctx, telem.TimeRange{Start: telem.SecondTS, End: telem.SecondTS * 5})
		Expect(svc.SetEnd(ctx, r.Key, telem.SecondTS*20)).To(Succeed())
		Expect(retrieve(ctx, r.Key).TimeRange.End).To(Equal(telem.SecondTS * 20))
	})

	It("Should return query.ErrNotFound when the range does not exist", func(ctx SpecContext) {
		Expect(svc.SetEnd(ctx, uuid.New(), telem.Now())).To(MatchError(query.ErrNotFound))
	})

	It("Should leave other fields untouched", func(ctx SpecContext) {
		r := create(ctx, telem.TimeRange{Start: telem.SecondTS, End: telem.TimeStampMax})
		Expect(svc.SetEnd(ctx, r.Key, telem.SecondTS*3)).To(Succeed())
		updated := retrieve(ctx, r.Key)
		Expect(updated.Name).To(Equal(r.Name))
		Expect(updated.Key).To(Equal(r.Key))
	})
})
