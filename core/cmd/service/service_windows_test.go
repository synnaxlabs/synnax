// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build windows

package service_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/synnax/cmd/service"
)

var _ = Describe("ParseServiceArgs", func() {
	BeforeEach(func() {
		viper.Reset()
	})

	Describe("key=value format", func() {
		It("should parse correctly", func() {
			Expect(service.ParseServiceArgs([]string{"--listen=0.0.0.0:9090"})).To(Succeed())
			Expect(viper.GetString("listen")).To(Equal("0.0.0.0:9090"))
		})
	})

	Describe("key value format", func() {
		It("should parse correctly", func() {
			Expect(service.ParseServiceArgs([]string{"--listen", "0.0.0.0:9090"})).To(Succeed())
			Expect(viper.GetString("listen")).To(Equal("0.0.0.0:9090"))
		})
	})

	Describe("boolean flag", func() {
		It("should parse correctly", func() {
			Expect(service.ParseServiceArgs([]string{"--insecure"})).To(Succeed())
			Expect(viper.GetString("insecure")).To(Equal("true"))
		})
	})

	Describe("multiple flags", func() {
		It("should parse all flags correctly", func() {
			Expect(service.ParseServiceArgs([]string{
				"--listen", "0.0.0.0:9090",
				"--insecure",
				"--data", "/path/to/data",
			})).To(Succeed())
			Expect(viper.GetString("listen")).To(Equal("0.0.0.0:9090"))
			Expect(viper.GetString("insecure")).To(Equal("true"))
			Expect(viper.GetString("data")).To(Equal("/path/to/data"))
		})
	})

	Describe("mixed formats", func() {
		It("should parse all formats correctly", func() {
			Expect(service.ParseServiceArgs([]string{
				"--listen=localhost:9090",
				"--insecure",
				"--data", "/data",
			})).To(Succeed())
			Expect(viper.GetString("listen")).To(Equal("localhost:9090"))
			Expect(viper.GetString("insecure")).To(Equal("true"))
			Expect(viper.GetString("data")).To(Equal("/data"))
		})
	})

	Describe("empty args", func() {
		It("should not error", func() {
			Expect(service.ParseServiceArgs([]string{})).To(Succeed())
		})
	})

	Describe("args without dashes", func() {
		It("should ignore non-flag arguments", func() {
			Expect(service.ParseServiceArgs([]string{"notaflag", "--listen", "0.0.0.0:9090"})).To(Succeed())
			Expect(viper.GetString("listen")).To(Equal("0.0.0.0:9090"))
		})
	})
})
