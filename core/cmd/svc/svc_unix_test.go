// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build !windows

package svc_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/cmd/svc"
)

var _ = Describe("Svc", func() {
	It("RegisterCommands", func() {
		It("should be a no-op on non-Windows platforms", func() {
			Expect(func() { svc.RegisterCommands(nil) }).ToNot(Panic())
		})
	})
})
