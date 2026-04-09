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
	"bytes"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/synnaxlabs/synnax/cmd/service"
	"github.com/synnaxlabs/synnax/pkg/version"
	"github.com/synnaxlabs/x/testutil"
)

var _ = Describe("FormatStatus", func() {
	It("Should include the version in the output", func() {
		cmd := &cobra.Command{}
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)

		info := service.StatusInfo{
			Installed:  true,
			State:      "Running",
			ProcessID:  12345,
			ConfigPath: `C:\ProgramData\Synnax\config.yaml`,
			DataDir:    `C:\ProgramData\Synnax\data`,
			Listen:     "localhost:9090",
			Insecure:   true,
		}
		service.FormatStatus(cmd, info)
		output := buf.String()
		Expect(output).To(ContainSubstring("Version: "))
		Expect(output).To(ContainSubstring(version.Get()))
	})

	It("Should show Not installed when service is not installed", func() {
		cmd := &cobra.Command{}
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)

		info := service.StatusInfo{ConfigPath: `C:\ProgramData\Synnax\config.yaml`}
		service.FormatStatus(cmd, info)
		Expect(buf.String()).To(ContainSubstring("Status:  Not installed"))
	})

	It("Should show PID when service is running", func() {
		cmd := &cobra.Command{}
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)

		info := service.StatusInfo{
			Installed:  true,
			State:      "Running",
			ProcessID:  42,
			ConfigPath: `C:\ProgramData\Synnax\config.yaml`,
		}
		service.FormatStatus(cmd, info)
		Expect(buf.String()).To(ContainSubstring("Status:  Running (PID: 42)"))
	})
})

var _ = Describe("WriteConfig", func() {
	var (
		tmpDir     string
		origPD     string
		configPath string
	)

	BeforeEach(func() {
		tmpDir = testutil.MustSucceed(os.MkdirTemp("", "synnax-service-test-*"))

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
		data := testutil.MustSucceed(os.ReadFile(configPath))
		Expect(string(data)).To(ContainSubstring("listen"))
	})

	It("Should preserve an existing config file on reinstall", func() {
		Expect(os.MkdirAll(filepath.Dir(configPath), 0755)).To(Succeed())
		originalContent := []byte("listen: localhost:9091\ncustom-key: custom-value\n")
		Expect(os.WriteFile(configPath, originalContent, 0644)).To(Succeed())

		viper.Set("listen", "localhost:9999")
		Expect(service.WriteConfig()).To(Succeed())

		data := testutil.MustSucceed(os.ReadFile(configPath))
		Expect(string(data)).To(Equal(string(originalContent)))
	})
})
