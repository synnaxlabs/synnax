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
	Describe("Validate", func() {
		It("should succeed if the encoder and content type are non-nil", func() {
			Expect(fhttp.DefaultClientConfig.Validate()).To(Succeed())
		})
		It("should return an error if the codec is nil", func() {
			cfg := fhttp.ClientConfig{Encoder: nil, ContentType: "this exists"}
			Expect(cfg.Validate()).To(MatchError(ContainSubstring("encoder")))
		})
		It("should return an error if the content type is empty", func() {
			cfg := fhttp.ClientConfig{Encoder: binary.JSONCodec, ContentType: ""}
			Expect(cfg.Validate()).To(MatchError(ContainSubstring("content_type")))
		})
	})
	Describe("Override", func() {
		var originalCfg fhttp.ClientConfig
		BeforeEach(func() {
			originalCfg = fhttp.ClientConfig{
				Encoder:     binary.GobCodec,
				ContentType: "application/x-gob",
			}
		})
		It("should override the codec if it is non-nil", func() {
			overrideCfg := fhttp.ClientConfig{Encoder: binary.JSONCodec}
			cfg := originalCfg.Override(overrideCfg)
			Expect(cfg.Encoder).To(Equal(overrideCfg.Encoder))
			Expect(cfg.ContentType).To(Equal(originalCfg.ContentType))
		})
		It("shouldn't override if the codec is nil", func() {
			overrideCfg := fhttp.ClientConfig{ContentType: "this exists"}
			cfg := originalCfg.Override(overrideCfg)
			Expect(cfg.Encoder).To(Equal(originalCfg.Encoder))
			Expect(cfg.ContentType).To(Equal(overrideCfg.ContentType))
		})
		It("should override the content type if it is non-empty", func() {
			overrideCfg := fhttp.ClientConfig{ContentType: "this exists"}
			cfg := originalCfg.Override(overrideCfg)
			Expect(cfg.Encoder).To(Equal(originalCfg.Encoder))
			Expect(cfg.ContentType).To(Equal(overrideCfg.ContentType))
		})
		It("shouldn't override if the content type is empty", func() {
			overrideCfg := fhttp.ClientConfig{Encoder: binary.MsgPackCodec}
			cfg := originalCfg.Override(overrideCfg)
			Expect(cfg.Encoder).To(Equal(overrideCfg.Encoder))
			Expect(cfg.ContentType).To(Equal(originalCfg.ContentType))
		})
	})
})
