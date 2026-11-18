// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build !driver

package driver_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/driver"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Build without the Driver", func() {
	It("Should report that the Core holds no Driver binary", func() {
		Expect(driver.Bundled()).To(BeFalse())
	})
	It("Should open a no-op Driver that needs no rack key", func(ctx SpecContext) {
		d := MustSucceed(driver.Open(ctx, driver.Config{
			Enabled:  new(true),
			Insecure: new(true),
		}))
		Expect(d.Close()).To(Succeed())
	})
})
