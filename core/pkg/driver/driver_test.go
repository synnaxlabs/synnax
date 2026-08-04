// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing/fstest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/driver"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// syncBuffer is a concurrency-safe log sink for tests that read the captured output
// while the driver's logging goroutines are still writing to it.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (*syncBuffer) Sync() error { return nil }

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func newTestLogger() (*alamos.Logger, *syncBuffer) {
	buffer := &syncBuffer{}
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

func openMockDriver(
	ctx context.Context,
	logger *alamos.Logger,
	overrides ...driver.Config,
) *driver.Driver {
	base := driver.Config{
		Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
		FS:              mockFS,
		Insecure:        new(true),
		Address:         "localhost:9090",
		ParentDirname:   GinkgoT().TempDir(),
	}
	cfgs := append([]driver.Config{base}, overrides...)
	return MustSucceed(driver.Open(ctx, cfgs...))
}

var _ = Describe("Open", func() {
	Describe("contract", func() {
		It("Should return a non-nil driver on success", func(ctx SpecContext) {
			logger, _ := newTestLogger()
			d := openMockDriver(ctx, logger)
			Expect(d).ToNot(BeNil())
			Expect(d.Close()).To(Succeed())
		})

		It(
			"Should return nil driver and kill the process promptly on timeout",
			func(ctx SpecContext) {
				Expect(os.Setenv("MOCK_DELAY_MS", "30000")).To(Succeed())
				defer func() { Expect(os.Unsetenv("MOCK_DELAY_MS")).To(Succeed()) }()
				logger, _ := newTestLogger()
				start := time.Now()
				d, err := driver.Open(ctx, driver.Config{
					Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
					FS:              mockFS,
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
			},
		)

		It(
			"Should return timeout error when driver crashes on startup",
			func(ctx SpecContext) {
				Expect(os.Setenv("MOCK_FAIL_START", "1")).To(Succeed())
				defer func() { Expect(os.Unsetenv("MOCK_FAIL_START")).To(Succeed()) }()
				logger, _ := newTestLogger()
				d, err := driver.Open(ctx, driver.Config{
					Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
					FS:              mockFS,
					Insecure:        new(true),
					Address:         "localhost:9090",
					ParentDirname:   GinkgoT().TempDir(),
					StartTimeout:    500 * time.Millisecond,
					StopTimeout:     500 * time.Millisecond,
				})
				Expect(err).To(MatchError(ContainSubstring("timed out")))
				Expect(d).To(BeNil())
			},
		)

		It(
			"Should return nil driver on config validation failure",
			func(ctx SpecContext) {
				d, err := driver.Open(ctx, driver.Config{})
				Expect(err).To(HaveOccurred())
				Expect(d).To(BeNil())
			},
		)

		It("Should return a non-nil driver when disabled", func(ctx SpecContext) {
			logger, _ := newTestLogger()
			d := MustSucceed(driver.Open(ctx, driver.Config{
				Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
				Enabled:         new(false),
				Insecure:        new(true),
			}))
			Expect(d).ToNot(BeNil())
			Expect(d.Close()).To(Succeed())
		})

		It(
			"Should return a non-nil no-op driver when the binary is unavailable",
			func(ctx SpecContext) {
				logger, buffer := newTestLogger()
				d := MustSucceed(driver.Open(ctx, driver.Config{
					Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
					Insecure:        new(true),
					Address:         "localhost:9090",
					ParentDirname:   GinkgoT().TempDir(),
				}))
				Expect(d.Close()).To(Succeed())
				Expect(buffer.String()).To(ContainSubstring(
					"Core built without embedded Driver",
				))
			},
		)

		It(
			"Should return an error when the restart policy config is invalid",
			func(ctx SpecContext) {
				logger, _ := newTestLogger()
				Expect(driver.Open(ctx, driver.Config{
					Instrumentation:   alamos.New("test", alamos.WithLogger(logger)),
					FS:                mockFS,
					Insecure:          new(true),
					Address:           "localhost:9090",
					ParentDirname:     GinkgoT().TempDir(),
					RestartMaxRetries: -1,
				})).Error().To(MatchError(ContainSubstring("max_retries")))
			},
		)

		It(
			"Should wrap the startup error when the context is canceled",
			func(ctx SpecContext) {
				canceledCtx, cancel := context.WithCancel(ctx)
				cancel()
				logger, _ := newTestLogger()
				Expect(driver.Open(canceledCtx, driver.Config{
					Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
					FS:              mockFS,
					Insecure:        new(true),
					Address:         "localhost:9090",
					ParentDirname:   GinkgoT().TempDir(),
					StartTimeout:    2 * time.Second,
					StopTimeout:     500 * time.Millisecond,
				})).Error().
					To(MatchError(ContainSubstring("failed to start embedded Driver")))
			},
		)
	})

	Describe("startup failures", func() {
		It(
			"Should fail to start when the binary is missing from the filesystem",
			func(ctx SpecContext) {
				logger, _ := newTestLogger()
				Expect(driver.Open(ctx, driver.Config{
					Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
					FS:              fstest.MapFS{},
					Insecure:        new(true),
					Address:         "localhost:9090",
					ParentDirname:   GinkgoT().TempDir(),
					StartTimeout:    200 * time.Millisecond,
					StopTimeout:     200 * time.Millisecond,
				})).Error().To(MatchError(ContainSubstring("timed out")))
			},
		)

		It(
			"Should fail to start when the working directory cannot be created",
			func(ctx SpecContext) {
				logger, _ := newTestLogger()
				blocker := filepath.Join(GinkgoT().TempDir(), "blocker")
				Expect(os.WriteFile(blocker, []byte("x"), 0o644)).To(Succeed())
				Expect(driver.Open(ctx, driver.Config{
					Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
					FS:              mockFS,
					Insecure:        new(true),
					Address:         "localhost:9090",
					ParentDirname:   filepath.Join(blocker, "sub"),
					StartTimeout:    200 * time.Millisecond,
					StopTimeout:     200 * time.Millisecond,
				})).Error().To(MatchError(ContainSubstring("timed out")))
			},
		)
	})

	Describe("behavior", func() {
		It("Should pass --debug flag when enabled", func(ctx SpecContext) {
			logger, buffer := newTestLogger()
			d := openMockDriver(ctx, logger, driver.Config{Debug: new(true)})
			Expect(d.Close()).To(Succeed())
			Expect(buffer.String()).To(ContainSubstring("debug mode enabled"))
		})
	})
})

var _ = Describe("restart", func() {
	It(
		"Should restart and recover after transient startup crashes",
		func(ctx SpecContext) {
			crashFile := filepath.Join(GinkgoT().TempDir(), "crashes")
			Expect(os.WriteFile(crashFile, []byte("2"), 0o644)).To(Succeed())
			Expect(os.Setenv("MOCK_CRASH_COUNT_FILE", crashFile)).To(Succeed())
			defer func() { Expect(os.Unsetenv("MOCK_CRASH_COUNT_FILE")).To(Succeed()) }()
			logger, buffer := newTestLogger()
			d := openMockDriver(ctx, logger, driver.Config{
				StartTimeout:        5 * time.Second,
				RestartBaseInterval: time.Millisecond,
				RestartMaxRetries:   10,
			})
			Expect(d).ToNot(BeNil())
			Expect(d.Close()).To(Succeed())
			Expect(strings.Count(buffer.String(), "exited unexpectedly")).
				To(BeNumerically(">=", 2))
		},
	)

	It("Should give up after exceeding the restart limit", func(ctx SpecContext) {
		crashFile := filepath.Join(GinkgoT().TempDir(), "crashes")
		Expect(os.WriteFile(crashFile, []byte("10"), 0o644)).To(Succeed())
		Expect(os.Setenv("MOCK_CRASH_COUNT_FILE", crashFile)).To(Succeed())
		defer func() { Expect(os.Unsetenv("MOCK_CRASH_COUNT_FILE")).To(Succeed()) }()
		logger, buffer := newTestLogger()
		d, err := driver.Open(ctx, driver.Config{
			Instrumentation:     alamos.New("test", alamos.WithLogger(logger)),
			FS:                  mockFS,
			Insecure:            new(true),
			Address:             "localhost:9090",
			ParentDirname:       GinkgoT().TempDir(),
			StartTimeout:        2 * time.Second,
			StopTimeout:         500 * time.Millisecond,
			RestartBaseInterval: time.Millisecond,
			RestartMaxRetries:   2,
		})
		Expect(err).To(MatchError(ContainSubstring("timed out")))
		Expect(d).To(BeNil())
		Expect(buffer.String()).To(ContainSubstring("exceeded restart limit"))
	})

	It("Should not restart after a graceful stop", func(ctx SpecContext) {
		logger, buffer := newTestLogger()
		d := openMockDriver(ctx, logger)
		Expect(d.Close()).To(Succeed())
		Expect(buffer.String()).ToNot(ContainSubstring("exited unexpectedly"))
	})

	It(
		"Should not panic when a restarted driver re-signals startup",
		func(ctx SpecContext) {
			Expect(os.Setenv("MOCK_EXIT_AFTER_MS", "100")).To(Succeed())
			defer func() { Expect(os.Unsetenv("MOCK_EXIT_AFTER_MS")).To(Succeed()) }()
			logger, buffer := newTestLogger()
			d := openMockDriver(ctx, logger, driver.Config{
				StartTimeout:        5 * time.Second,
				RestartBaseInterval: time.Millisecond,
			})
			Expect(d).ToNot(BeNil())
			// Each run starts then crashes, so the supervisor restarts and the
			// restarted run re-emits "started successfully". The shared startup channel
			// must not be closed twice: on the bug this panics in the pipe goroutine
			// before the second startup line is logged, so the count never reaches two.
			Eventually(func() int {
				return strings.Count(buffer.String(), "started successfully")
			}, 5*time.Second).Should(BeNumerically(">=", 2))
			Expect(d.Close()).To(Succeed())
			Expect(buffer.String()).ToNot(ContainSubstring("close of closed channel"))
		},
	)
})

var _ = Describe("Close", func() {
	Describe("contract", func() {
		It("Should shut down cleanly after successful Open", func(ctx SpecContext) {
			logger, _ := newTestLogger()
			d := openMockDriver(ctx, logger)
			Expect(d.Close()).To(Succeed())
		})

		It("Should be idempotent on an enabled driver", func(ctx SpecContext) {
			logger, _ := newTestLogger()
			d := openMockDriver(ctx, logger)
			Expect(d.Close()).To(Succeed())
			Expect(d.Close()).To(Succeed())
		})

		It(
			"Should handle a driver crash after successful startup",
			func(ctx SpecContext) {
				Expect(os.Setenv("MOCK_EXIT_AFTER_MS", "100")).To(Succeed())
				defer func() { Expect(os.Unsetenv("MOCK_EXIT_AFTER_MS")).To(Succeed()) }()
				logger, _ := newTestLogger()
				d := openMockDriver(ctx, logger)
				time.Sleep(200 * time.Millisecond)
				Expect(d.Close()).To(Succeed())
			},
		)

		It("Should be idempotent on a disabled driver", func(ctx SpecContext) {
			logger, _ := newTestLogger()
			d := MustSucceed(driver.Open(ctx, driver.Config{
				Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
				Enabled:         new(false),
				Insecure:        new(true),
			}))
			Expect(d.Close()).To(Succeed())
			Expect(d.Close()).To(Succeed())
		})
	})
})
