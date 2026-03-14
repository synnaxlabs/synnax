// Copyright 2026 Synnax Labs, Inc.
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
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"

	"github.com/synnaxlabs/synnax/cmd/service"
)

var _ = Describe("WriteConfig", func() {
	var (
		tmpDir      string
		origPD      string
		configPath  string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "synnax-service-test-*")
		Expect(err).ToNot(HaveOccurred())

		origPD = os.Getenv("ProgramData")
		Expect(os.Setenv("ProgramData", tmpDir)).To(Succeed())

		configPath = filepath.Join(tmpDir, "Synnax", "config.yaml")

		viper.Reset()
		viper.Set("listen", "localhost:9090")
		viper.Set("insecure", true)
	})

	AfterEach(func() {
		Expect(os.Setenv("ProgramData", origPD)).To(Succeed())
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
		viper.Reset()
	})

	It("Should create a config file on fresh install", func() {
		Expect(service.WriteConfig()).To(Succeed())
		Expect(configPath).To(BeAnExistingFile())
		data, err := os.ReadFile(configPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(data)).To(ContainSubstring("listen"))
	})

	It("Should preserve an existing config file on reinstall", func() {
		Expect(os.MkdirAll(filepath.Dir(configPath), 0755)).To(Succeed())
		originalContent := []byte("listen: localhost:9091\ncustom-key: custom-value\n")
		Expect(os.WriteFile(configPath, originalContent, 0644)).To(Succeed())

		viper.Set("listen", "localhost:9999")
		Expect(service.WriteConfig()).To(Succeed())

		data, err := os.ReadFile(configPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(data)).To(Equal(string(originalContent)))
	})
})
