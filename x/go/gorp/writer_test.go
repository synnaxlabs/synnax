// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/gorp"
)

var _ = Describe("Writer", func() {
	It("Should wrap a key-value writer with an encoder", func() {
		tx := db.OpenTx()
		w := gorp.WrapWriter[int32, entry](tx)
		ctx := context.Background()
		Expect(w.Set(ctx, entry{ID: 1, Data: "Two"})).To(Succeed())
		Expect(w.Delete(ctx, 1)).To(Succeed())
	})
})
