// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build driver

package driver_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/driver"
	"github.com/synnaxlabs/x/config"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func newTestLogger() (*alamos.Logger, *bytes.Buffer) {
	buffer := &bytes.Buffer{}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.AddSync(buffer),
		zapcore.DebugLevel,
	)
	logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
		ZapLogger: zap.New(core),
	}))
	return logger, buffer
}

func openMockDriver(logger *alamos.Logger, overrides ...driver.Config) *driver.Driver {
	base := driver.Config{
		Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
		BinaryPath:      mockBinaryPath,
		Insecure:        config.True(),
		Address:         "localhost:9090",
		ParentDirname:   GinkgoT().TempDir(),
	}
	cfgs := append([]driver.Config{base}, overrides...)
	return MustSucceed(driver.Open(context.Background(), cfgs...))
}

var _ = Describe("Open", func() {
	Describe("contract", func() {
		It("Should return a non-nil driver on success", func() {
			logger, _ := newTestLogger()
			d := openMockDriver(logger)
			Expect(d).ToNot(BeNil())
			Expect(d.Close()).To(Succeed())
		})

		It("Should return nil driver on timeout", func() {
			Expect(os.Setenv("MOCK_DELAY_MS", "5000")).To(Succeed())
			defer func() { Expect(os.Unsetenv("MOCK_DELAY_MS")).To(Succeed()) }()
			logger, _ := newTestLogger()
			d, err := driver.Open(context.Background(), driver.Config{
				Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
				BinaryPath:      mockBinaryPath,
				Insecure:        config.True(),
				Address:         "localhost:9090",
				ParentDirname:   GinkgoT().TempDir(),
				StartTimeout:    100 * time.Millisecond,
			})
			Expect(err).To(MatchError(ContainSubstring("timed out")))
			Expect(d).To(BeNil())
		})

		It("Should not leak processes after timeout", func() {
			Expect(os.Setenv("MOCK_DELAY_MS", "5000")).To(Succeed())
			defer func() { Expect(os.Unsetenv("MOCK_DELAY_MS")).To(Succeed()) }()
			logger, _ := newTestLogger()
			d, err := driver.Open(context.Background(), driver.Config{
				Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
				BinaryPath:      mockBinaryPath,
				Insecure:        config.True(),
				Address:         "localhost:9090",
				ParentDirname:   GinkgoT().TempDir(),
				StartTimeout:    100 * time.Millisecond,
			})
			Expect(err).To(HaveOccurred())
			Expect(d).To(BeNil())
			Eventually(func() bool {
				out, _ := exec.Command("pgrep", "-f", "mockdriver").Output()
				for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
					if line == "" {
						continue
					}
					pid := MustSucceed(strconv.Atoi(line))
					process, err := os.FindProcess(pid)
					if err != nil {
						continue
					}
					if err := process.Signal(os.Signal(nil)); err == nil {
						return false
					}
				}
				return true
			}, 10*time.Second, 100*time.Millisecond).Should(BeTrue())
		})

		It("Should return nil driver on config validation failure", func() {
			d, err := driver.Open(context.Background(), driver.Config{})
			Expect(err).To(HaveOccurred())
			Expect(d).To(BeNil())
		})

		It("Should return a non-nil driver when disabled", func() {
			logger, _ := newTestLogger()
			d := MustSucceed(driver.Open(context.Background(), driver.Config{
				Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
				Enabled:         config.False(),
				Insecure:        config.True(),
			}))
			Expect(d).ToNot(BeNil())
			Expect(d.Close()).To(Succeed())
		})
	})

	Describe("behavior", func() {
		It("Should pass --debug flag when enabled", func() {
			logger, buffer := newTestLogger()
			d := openMockDriver(logger, driver.Config{Debug: config.True()})
			Expect(buffer.String()).To(ContainSubstring("debug mode enabled"))
			Expect(d.Close()).To(Succeed())
		})
	})
})

var _ = Describe("Close", func() {
	Describe("contract", func() {
		It("Should shut down cleanly after successful Open", func() {
			logger, _ := newTestLogger()
			d := openMockDriver(logger)
			Expect(d.Close()).To(Succeed())
		})

		It("Should be idempotent on an enabled driver", func() {
			logger, _ := newTestLogger()
			d := openMockDriver(logger)
			Expect(d.Close()).To(Succeed())
			Expect(d.Close()).To(Succeed())
		})

		It("Should be idempotent on a disabled driver", func() {
			logger, _ := newTestLogger()
			d := MustSucceed(driver.Open(context.Background(), driver.Config{
				Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
				Enabled:         config.False(),
				Insecure:        config.True(),
			}))
			Expect(d.Close()).To(Succeed())
			Expect(d.Close()).To(Succeed())
		})
	})
})
