// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Retrieve", func() {
	Describe("ByKey", func() {
		var a, b icd.ICD
		BeforeEach(func(ctx SpecContext) {
			a, b = icd.ICD{Name: "A"}, icd.ICD{Name: "B"}
			w := svc.NewWriter(tx)
			Expect(w.Create(ctx, &a)).To(Succeed())
			Expect(w.Create(ctx, &b)).To(Succeed())
		})

		It("Should return each ICD that it finds by its key", func(ctx SpecContext) {
			icds := MustSucceed(svc.NewRetrieve().
				Where(icd.MatchKeys(a.Key, b.Key)).
				ByKey(ctx, tx))
			Expect(icds).To(HaveLen(2))
			Expect(icds).To(HaveKeyWithValue(a.Key, HaveField("Name", "A")))
			Expect(icds).To(HaveKeyWithValue(b.Key, HaveField("Name", "B")))
		})

		It("Should fail when a key does not exist", func(ctx SpecContext) {
			Expect(svc.NewRetrieve().
				Where(icd.MatchKeys(a.Key, uuid.New())).
				ByKey(ctx, tx)).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should leave out a key that does not exist when it skips missing keys",
			func(ctx SpecContext) {
				icds := MustSucceed(svc.NewRetrieve().
					Where(icd.MatchKeys(a.Key, uuid.New())).
					SkipMissingKeys().
					ByKey(ctx, tx))
				Expect(icds).To(HaveLen(1))
				Expect(icds).To(HaveKeyWithValue(a.Key, HaveField("Name", "A")))
			},
		)
	})
})
