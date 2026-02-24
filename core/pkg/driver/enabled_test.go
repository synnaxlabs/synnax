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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/driver"
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
		Insecure:        new(true),
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

		It("Should return nil driver and kill the process promptly on timeout", func() {
			Expect(os.Setenv("MOCK_DELAY_MS", "30000")).To(Succeed())
			defer func() { Expect(os.Unsetenv("MOCK_DELAY_MS")).To(Succeed()) }()
			logger, _ := newTestLogger()
			start := time.Now()
			d, err := driver.Open(context.Background(), driver.Config{
				Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
				BinaryPath:      mockBinaryPath,
				Insecure:        new(true),
				Address:         "localhost:9090",
				ParentDirname:   GinkgoT().TempDir(),
				StartTimeout:    100 * time.Millisecond,
				StopTimeout:     500 * time.Millisecond,
			})
			elapsed := time.Since(start)
			Expect(err).To(MatchError(ContainSubstring("timed out")))
			Expect(d).To(BeNil())
			// Open should return within StartTimeout + StopTimeout + some
			// margin. With a 30s mock delay, exceeding 5s here would indicate
			// the kill escalation is broken.
			Expect(elapsed).To(BeNumerically("<", 5*time.Second))
		})

		It("Should return timeout error when driver crashes on startup", func() {
			Expect(os.Setenv("MOCK_FAIL_START", "1")).To(Succeed())
			defer func() { Expect(os.Unsetenv("MOCK_FAIL_START")).To(Succeed()) }()
			logger, _ := newTestLogger()
			d, err := driver.Open(context.Background(), driver.Config{
				Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
				BinaryPath:      mockBinaryPath,
				Insecure:        new(true),
				Address:         "localhost:9090",
				ParentDirname:   GinkgoT().TempDir(),
				StartTimeout:    500 * time.Millisecond,
				StopTimeout:     500 * time.Millisecond,
			})
			Expect(err).To(MatchError(ContainSubstring("timed out")))
			Expect(d).To(BeNil())
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
				Enabled:         new(false),
				Insecure:        new(true),
			}))
			Expect(d).ToNot(BeNil())
			Expect(d.Close()).To(Succeed())
		})
	})

	Describe("behavior", func() {
		It("Should pass --debug flag when enabled", func() {
			logger, buffer := newTestLogger()
			d := openMockDriver(logger, driver.Config{Debug: new(true)})
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

		It("Should handle a driver crash after successful startup", func() {
			Expect(os.Setenv("MOCK_EXIT_AFTER_MS", "100")).To(Succeed())
			defer func() { Expect(os.Unsetenv("MOCK_EXIT_AFTER_MS")).To(Succeed()) }()
			logger, _ := newTestLogger()
			d := openMockDriver(logger)
			time.Sleep(200 * time.Millisecond)
			Expect(d.Close()).To(Succeed())
		})

		It("Should be idempotent on a disabled driver", func() {
			logger, _ := newTestLogger()
			d := MustSucceed(driver.Open(context.Background(), driver.Config{
				Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
				Enabled:         new(false),
				Insecure:        new(true),
			}))
			Expect(d.Close()).To(Succeed())
			Expect(d.Close()).To(Succeed())
		})
	})
})
