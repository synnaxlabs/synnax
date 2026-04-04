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
	"github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap/zapcore"
)

var _ = Describe("Instrumentation", func() {
	Describe("ObservedInstrumentation", func() {
		It("Should capture log entries at the configured level", func() {
			ins, logs := testutil.ObservedInstrumentation(zapcore.InfoLevel)
			ins.L.Info("hello")
			Expect(logs.Len()).To(Equal(1))
			Expect(logs.All()[0].Message).To(Equal("hello"))
		})

		It("Should not capture entries below the configured level", func() {
			ins, logs := testutil.ObservedInstrumentation(zapcore.WarnLevel)
			ins.L.Info("ignored")
			ins.L.Warn("captured")
			Expect(logs.Len()).To(Equal(1))
			Expect(logs.All()[0].Message).To(Equal("captured"))
		})

		It("Should return a non-zero instrumentation", func() {
			ins, _ := testutil.ObservedInstrumentation(zapcore.InfoLevel)
			Expect(ins.IsZero()).To(BeFalse())
		})
	})

	Describe("PanicLogger", func() {
		It("Should return a non-zero instrumentation", func() {
			ins := testutil.PanicLogger()
			Expect(ins.IsZero()).To(BeFalse())
		})
	})
})
