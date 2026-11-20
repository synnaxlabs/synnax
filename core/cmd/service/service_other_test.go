// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build !linux && !windows

package service_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/cmd/service"
)

var _ = Describe("Service", func() {
	It("Should find no service", func() {
		Expect(service.Lookup()).Error().ToNot(HaveOccurred())
	})

	It("Should not start or stop a service", func() {
		const unsupported = "the Core service runs only on Linux and Windows"
		Expect(service.Start()).To(MatchError(unsupported))
		Expect(service.Stop()).To(MatchError(unsupported))
	})
})
