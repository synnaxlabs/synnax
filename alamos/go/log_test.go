// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alamos_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
)

var _ = Describe("Log", func() {
	Describe("NewLogger", func() {
		It("Should correctly attach a new logger to the Instrumentation", func() {
			logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{ZapConfig: zap.NewDevelopmentConfig()}))
			i := alamos.New("test", alamos.WithLogger(logger))
			Expect(i.L).ToNot(BeNil())
		})
	})
	Describe("No-op", func() {
		It("Should not panic when calling a method on a nil logger", func() {
			var l *alamos.Logger
			Expect(func() { l.Debug("test") }).ToNot(Panic())
		})
	})
})
