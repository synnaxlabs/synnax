// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/storage/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("OpenLayer", func() {
	ShouldNotLeakGoroutinesPerSpec()
	It("Should open an in-memory storage layer", func(ctx SpecContext) {
		l := testutil.OpenLayer(ctx)
		Expect(l.KV.Set(ctx, []byte("foo"), []byte("bar"))).To(Succeed())
		value, closer := MustSucceed2(l.KV.Get(ctx, []byte("foo")))
		Expect(value).To(Equal([]byte("bar")))
		Expect(closer.Close()).To(Succeed())
		Expect(l.Close()).To(Succeed())
	})
})
