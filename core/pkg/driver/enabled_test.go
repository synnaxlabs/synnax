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
	"github.com/synnaxlabs/x/config"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ = Describe("Lifecycle", func() {
	var (
		logger *alamos.Logger
		buffer *bytes.Buffer
	)

	BeforeEach(func() {
		buffer = &bytes.Buffer{}
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()),
			zapcore.AddSync(buffer),
			zapcore.DebugLevel,
		)
		logger = MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
			ZapLogger: zap.New(core),
		}))
	})

	It("Should successfully start and signal started", func() {
		tmpDir := GinkgoT().TempDir()
		d := MustSucceed(driver.Open(context.Background(), driver.Config{
			Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
			BinaryPath:      mockBinaryPath,
			Insecure:        config.True(),
			Address:         "localhost:9090",
			ParentDirname:   tmpDir,
		}))
		Expect(d.Close()).To(Succeed())
	})

	It("Should shut down cleanly via Close", func() {
		tmpDir := GinkgoT().TempDir()
		d := MustSucceed(driver.Open(context.Background(), driver.Config{
			Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
			BinaryPath:      mockBinaryPath,
			Insecure:        config.True(),
			Address:         "localhost:9090",
			ParentDirname:   tmpDir,
		}))
		Expect(d.Close()).To(Succeed())
	})

	It("Should return timeout error on slow start", func() {
		os.Setenv("MOCK_DELAY_MS", "5000")
		defer os.Unsetenv("MOCK_DELAY_MS")
		tmpDir := GinkgoT().TempDir()
		d, err := driver.Open(context.Background(), driver.Config{
			Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
			BinaryPath:      mockBinaryPath,
			Insecure:        config.True(),
			Address:         "localhost:9090",
			ParentDirname:   tmpDir,
			StartTimeout:    100 * time.Millisecond,
		})
		Expect(err).To(MatchError(ContainSubstring("timed out")))
		if d != nil {
			d.Close()
		}
	})

	It("Should pass --debug flag when enabled", func() {
		tmpDir := GinkgoT().TempDir()
		d := MustSucceed(driver.Open(context.Background(), driver.Config{
			Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
			BinaryPath:      mockBinaryPath,
			Insecure:        config.True(),
			Address:         "localhost:9090",
			ParentDirname:   tmpDir,
			Debug:           config.True(),
		}))
		Expect(buffer.String()).To(ContainSubstring("debug mode enabled"))
		Expect(d.Close()).To(Succeed())
	})

	It("Should no-op when disabled", func() {
		d := MustSucceed(driver.Open(context.Background(), driver.Config{
			Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
			Enabled:         config.False(),
			Insecure:        config.True(),
		}))
		Expect(d.Close()).To(Succeed())
	})

	It("Should close safely when disabled", func() {
		d := MustSucceed(driver.Open(context.Background(), driver.Config{
			Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
			Enabled:         config.False(),
			Insecure:        config.True(),
		}))
		Expect(d.Close()).To(Succeed())
		Expect(d.Close()).To(Succeed())
	})
})
