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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
)

var _ = Describe("Service", func() {
	Describe("Observe", func() {
		It("Should notify a handler of the ICDs that a commit changed",
			func(ctx SpecContext) {
				var changed []icd.Key
				DeferCleanup(svc.Observe().OnChange(func(
					_ context.Context,
					r gorp.TxReader[icd.Key, icd.ICD],
				) {
					for ch := range r {
						changed = append(changed, ch.Key)
					}
				}))
				ic := icd.ICD{Name: "Engine"}
				Expect(svc.NewWriter(tx).Create(ctx, &ic)).To(Succeed())
				Expect(tx.Commit(ctx)).To(Succeed())
				Expect(changed).To(Equal([]icd.Key{ic.Key}))
			},
		)
	})
})
