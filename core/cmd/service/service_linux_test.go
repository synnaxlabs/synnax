// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build linux

package service_test

import (
	"bytes"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"github.com/synnaxlabs/synnax/cmd/service"
	servicetest "github.com/synnaxlabs/synnax/cmd/service/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	var systemctl *servicetest.Systemctl
	BeforeEach(func() { systemctl = servicetest.NewSystemctl() })

	Describe("Lookup", func() {
		It("Should find no service when none is installed", func() {
			Expect(service.Lookup()).Error().ToNot(HaveOccurred())
		})

		It("Should find no service without systemd", func() {
			GinkgoT().Setenv("PATH", GinkgoT().TempDir())
			Expect(service.Lookup()).Error().ToNot(HaveOccurred())
		})

		It("Should find no service when systemd does not run", func() {
			systemctl.Install("/var/lib/synnax", "/usr/local/bin/synnax", "start")
			systemctl.Offline()
			_, installed := MustSucceed2(service.Lookup())
			Expect(installed).To(BeFalse())
		})

		It("Should return no service when the config file fails to read", func() {
			config := filepath.Join(GinkgoT().TempDir(), "config.yaml")
			systemctl.Install(
				"/var/lib/synnax",
				"/usr/local/bin/synnax",
				"start",
				"--config",
				config,
			)
			inst, installed, err := service.Lookup()
			Expect(err).To(MatchError(
				HavePrefix("failed to read service configuration"),
			))
			Expect(inst).To(BeZero())
			Expect(installed).To(BeFalse())
		})

		DescribeTable(
			"Should read the data directory from the service command",
			func(args []string, data string) {
				systemctl.Install("/var/lib/synnax", "/usr/local/bin/synnax", args...)
				inst, installed := MustSucceed2(service.Lookup())
				Expect(installed).To(BeTrue())
				Expect(inst).To(Equal(service.Installation{
					Binary: "/usr/local/bin/synnax",
					Data:   data,
				}))
			},
			Entry(
				"flag with a value",
				[]string{"start", "--listen=localhost:9090", "--data=/mnt/ssd1"},
				"/mnt/ssd1",
			),
			Entry(
				"flag before its value",
				[]string{"start", "--listen", "localhost:9090", "-d", "/mnt/ssd1"},
				"/mnt/ssd1",
			),
			Entry(
				"relative to the working directory",
				[]string{"start", "--data=data"},
				"/var/lib/synnax/data",
			),
			Entry(
				"path with a space",
				[]string{"start", "--data", "/mnt/my data"},
				"/mnt/my data",
			),
			Entry("default", []string{"start"}, "/var/lib/synnax/synnax-data"),
		)

		It("Should read the data directory from the config file", func() {
			config := filepath.Join(GinkgoT().TempDir(), "config.yaml")
			Expect(os.WriteFile(config, []byte("data: /mnt/ssd2\n"), 0o600)).
				To(Succeed())
			systemctl.Install(
				"/var/lib/synnax",
				"/usr/local/bin/synnax",
				"start",
				"--config",
				config,
			)
			inst, _ := MustSucceed2(service.Lookup())
			Expect(inst.Data).To(Equal("/mnt/ssd2"))
		})
	})

	It("Should start and stop the service", func() {
		Expect(service.Start()).To(Succeed())
		Expect(service.Stop()).To(Succeed())
		Expect(systemctl.Calls()).To(Equal([]string{"start synnax", "stop synnax"}))
	})

	Describe("FormatStatus", func() {
		format := func(info service.StatusInfo) string {
			cmd := &cobra.Command{}
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			service.FormatStatus(cmd, info)
			return buf.String()
		}

		It("Should show a running service with its PID", func() {
			Expect(format(service.StatusInfo{
				Installed:  true,
				State:      "active (running)",
				ProcessID:  42,
				ConfigPath: "/etc/synnax/config.yaml",
				DataDir:    "/var/lib/synnax/data",
			})).To(SatisfyAll(
				ContainSubstring("Status:  active (running) (PID: 42)"),
				ContainSubstring("Data dir:     /var/lib/synnax/data"),
			))
		})

		It("Should show the exit status of a failed service", func() {
			Expect(format(service.StatusInfo{
				Installed:  true,
				State:      "failed (failed)",
				ExitStatus: 1,
				ConfigPath: "/etc/synnax/config.yaml",
			})).To(ContainSubstring("Exit:    Status 1"))
		})

		It("Should show a service that is not installed", func() {
			Expect(format(service.StatusInfo{ConfigPath: "/etc/synnax/config.yaml"})).
				To(ContainSubstring("Status:  Not installed"))
		})
	})
})
