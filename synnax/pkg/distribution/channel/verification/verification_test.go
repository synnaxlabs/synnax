// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package verification_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel/verification"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Verification", func() {
	var db kv.DB
	var ctx context.Context
	BeforeEach(func() {
		db = memkv.New()
		ctx = context.Background()
	})
	AfterEach(func() {
		Expect(db.Close()).To(Succeed())
	})
	Describe("Service", func() {
		It("should open without a license key", func() {
			svc := MustSucceed(verification.OpenService(ctx, verification.Config{DB: db}))
			Expect(svc).ToNot(BeNil())
			Expect(svc.Close()).To(Succeed())
		})
	})
})
