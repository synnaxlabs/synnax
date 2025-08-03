// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fhttp_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter/fhttp"
	"github.com/synnaxlabs/x/binary"
)

var _ = Describe("ClientConfig", func() {
	var (
		jsonConfig    fhttp.ClientConfig
		msgpackConfig fhttp.ClientConfig
		nilConfig     fhttp.ClientConfig
	)
	BeforeEach(func() {
		jsonConfig = fhttp.ClientConfig{Codec: binary.JSONCodec}
		msgpackConfig = fhttp.ClientConfig{Codec: binary.MsgPackCodec}
		nilConfig = fhttp.ClientConfig{Codec: nil}
	})
	Describe("Validate", func() {
		It("should succeed if the codec is non-nil", func() {
			Expect(fhttp.DefaultClientConfig.Validate()).To(Succeed())
			Expect(jsonConfig.Validate()).To(Succeed())
			Expect(msgpackConfig.Validate()).To(Succeed())
		})
		It("should return an error if the codec is nil", func() {
			Expect(nilConfig.Validate()).To(MatchError(ContainSubstring("codec")))
		})
	})
	Describe("Override", func() {
		It("should override the codec", func() {
			Expect(jsonConfig.Override(msgpackConfig)).To(Equal(msgpackConfig))
		})
		It("shouldn't override if the codec is nil", func() {
			Expect(jsonConfig.Override(nilConfig)).To(Equal(jsonConfig))
		})
	})
})
