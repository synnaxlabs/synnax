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
	"encoding/json/v2"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing/fstest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/driver"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	xfs "github.com/synnaxlabs/x/io/fs"
	fstestutil "github.com/synnaxlabs/x/io/fs/testutil"
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

func (b *syncBuffer) Sync() error { return nil }

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func newTestLogger() (*alamos.Logger, *syncBuffer) {
	GinkgoHelper()
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

var rackKey = rack.NewKey(1, 1)

// mockConfig returns a config that starts the mock Driver in new directories.
func mockConfig(logger *alamos.Logger) driver.Config {
	return driver.Config{
		Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
		FS:              mockFS,
		Insecure:        new(true),
		Address:         "localhost:9090",
		ParentDirname:   GinkgoT().TempDir(),
		BinaryDirname:   GinkgoT().TempDir(),
		Keys:            keys,
		RackKey:         rackKey,
	}
}

func openMockDriver(
	ctx context.Context,
	logger *alamos.Logger,
	overrides ...driver.Config,
) *driver.Driver {
	GinkgoHelper()
	cfgs := append([]driver.Config{mockConfig(logger)}, overrides...)
	return MustSucceed(driver.Open(ctx, cfgs...))
}

// racingFS reports a failed rename after it renames, as when another Core writes the
// same binary first and the rename onto it fails.
type racingFS struct{ xfs.FS }

func (r racingFS) Sub(name string) (xfs.FS, error) {
	sub, err := r.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return racingFS{FS: sub}, nil
}

func (r racingFS) Rename(oldPath, newPath string) error {
	if err := r.FS.Rename(oldPath, newPath); err != nil {
		return err
	}
	return fstestutil.ErrFault
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
					Keys:            keys,
					RackKey:         rackKey,
					ParentDirname:   GinkgoT().TempDir(),
					BinaryDirname:   GinkgoT().TempDir(),
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
			"Should return before the Driver starts when detached",
			func(ctx SpecContext) {
				Expect(os.Setenv("MOCK_DELAY_MS", "30000")).To(Succeed())
				defer func() { Expect(os.Unsetenv("MOCK_DELAY_MS")).To(Succeed()) }()
				logger, _ := newTestLogger()
				start := time.Now()
				d := openMockDriver(ctx, logger, driver.Config{
					Detached:     new(true),
					StartTimeout: 10 * time.Second,
					StopTimeout:  500 * time.Millisecond,
				})
				// An attached open would block for the whole start timeout.
				Expect(time.Since(start)).To(BeNumerically("<", 5*time.Second))
				Expect(d.Close()).To(Succeed())
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
					Keys:            keys,
					RackKey:         rackKey,
					ParentDirname:   GinkgoT().TempDir(),
					BinaryDirname:   GinkgoT().TempDir(),
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
				Expect(
					err,
				).To(MatchError(ContainSubstring("insecure: must be non-nil")))
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
					Keys:            keys,
					RackKey:         rackKey,
					ParentDirname:   GinkgoT().TempDir(),
					BinaryDirname:   GinkgoT().TempDir(),
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
					Instrumentation:     alamos.New("test", alamos.WithLogger(logger)),
					FS:                  mockFS,
					Insecure:            new(true),
					Address:             "localhost:9090",
					Keys:                keys,
					RackKey:             rackKey,
					ParentDirname:       GinkgoT().TempDir(),
					BinaryDirname:       GinkgoT().TempDir(),
					RestartBaseInterval: -time.Second,
				})).Error().To(MatchError(ContainSubstring("base_interval")))
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
					Keys:            keys,
					RackKey:         rackKey,
					ParentDirname:   GinkgoT().TempDir(),
					BinaryDirname:   GinkgoT().TempDir(),
					StartTimeout:    2 * time.Second,
					StopTimeout:     500 * time.Millisecond,
				})).Error().
					To(MatchError(ContainSubstring("failed to start embedded Driver")))
			},
		)
	})

	Describe("startup failures", func() {
		DescribeTable(
			"Should fail to start when the binary cannot be extracted",
			func(ctx SpecContext, opt fstestutil.FaultyFSOption) {
				logger, _ := newTestLogger()
				binDir := GinkgoT().TempDir()
				binFS := fstestutil.WrapFaultyFS(xfs.Default, opt)
				Expect(driver.Open(ctx, mockConfig(logger), driver.Config{
					BinaryDirname: binDir,
					BinaryFS:      binFS,
				})).Error().To(MatchError(fstestutil.ErrFault))
				Expect(binFS.OpenFiles()).To(BeZero())
				Expect(os.ReadDir(binDir)).To(BeEmpty())
			},
			Entry("stat", fstestutil.WithFailStat()),
			Entry("create", fstestutil.WithFailOpen()),
			Entry("write", fstestutil.WithFailWrite()),
			Entry("rename", fstestutil.WithFailRename()),
		)

		It(
			"Should fail to start when the binary is missing from the filesystem",
			func(ctx SpecContext) {
				logger, _ := newTestLogger()
				Expect(driver.Open(ctx, mockConfig(logger), driver.Config{
					FS: fstest.MapFS{},
				})).Error().To(MatchError(fs.ErrNotExist))
			},
		)

		It(
			"Should fail to start when the binary directory cannot be created",
			func(ctx SpecContext) {
				logger, _ := newTestLogger()
				blocker := filepath.Join(GinkgoT().TempDir(), "blocker")
				Expect(os.WriteFile(blocker, []byte("x"), 0o644)).To(Succeed())
				Expect(driver.Open(ctx, mockConfig(logger), driver.Config{
					BinaryDirname: filepath.Join(blocker, "sub"),
				})).Error().To(MatchError(syscall.ENOTDIR))
			},
		)

		It(
			"Should fail to start when the working directory cannot be created",
			func(ctx SpecContext) {
				logger, _ := newTestLogger()
				blocker := filepath.Join(GinkgoT().TempDir(), "blocker")
				Expect(os.WriteFile(blocker, []byte("x"), 0o644)).To(Succeed())
				Expect(driver.Open(ctx, mockConfig(logger), driver.Config{
					ParentDirname: filepath.Join(blocker, "sub"),
				})).Error().To(MatchError(syscall.ENOTDIR))
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

		It("Should point the Driver at the trust anchors it is given", func(
			ctx SpecContext,
		) {
			logger, _ := newTestLogger()
			dir := GinkgoT().TempDir()
			envFile := filepath.Join(GinkgoT().TempDir(), "env")
			Expect(os.Setenv("MOCK_ENV_DUMP_FILE", envFile)).To(Succeed())
			defer func() { Expect(os.Unsetenv("MOCK_ENV_DUMP_FILE")).To(Succeed()) }()
			anchors := []byte("-----BEGIN CERTIFICATE-----\nanchors\n")
			d := openMockDriver(ctx, logger, driver.Config{
				Insecure:        new(false),
				ParentDirname:   dir,
				TrustAnchorsPEM: anchors,
			})
			Expect(readDriverConnection(dir)).To(HaveKeyWithValue("secure", true))
			anchorFile := filepath.Join(dir, "driver", "trust-anchors.pem")
			Expect(os.ReadFile(anchorFile)).To(Equal(anchors))
			Expect(os.ReadFile(envFile)).To(ContainSubstring(
				"GRPC_DEFAULT_SSL_ROOTS_FILE_PATH=" + anchorFile,
			))
			Expect(d.Close()).To(Succeed())
		})

		It("Should give the Driver an API key for its rack", func(ctx SpecContext) {
			logger, _ := newTestLogger()
			dir := GinkgoT().TempDir()
			d := openMockDriver(ctx, logger, driver.Config{ParentDirname: dir})
			conn := readDriverConnection(dir)
			Expect(d.Close()).To(Succeed())
			Expect(conn).ToNot(HaveKey("username"))
			Expect(conn).ToNot(HaveKey("password"))
			Expect(authenticate(ctx, conn["api_key"].(string)).Subject).
				To(Equal(rackKey.OntologyID()))
		})

		It("Should reuse its API key across Core processes", func(ctx SpecContext) {
			logger, _ := newTestLogger()
			keyDir := GinkgoT().TempDir()
			start := func() string {
				GinkgoHelper()
				Expect(openMockDriver(ctx, logger, driver.Config{
					ParentDirname: GinkgoT().TempDir(),
					BinaryDirname: GinkgoT().TempDir(),
					KeyDirname:    keyDir,
				}).Close()).To(Succeed())
				return readSavedKey(keyDir)
			}
			Expect(start()).To(Equal(start()))
		})

		It("Should create a new API key when the Core refuses the saved one", func(
			ctx SpecContext,
		) {
			logger, _ := newTestLogger()
			keyDir := GinkgoT().TempDir()
			cfg := driver.Config{KeyDirname: keyDir}
			Expect(openMockDriver(ctx, logger, cfg).Close()).To(Succeed())
			first := readSavedKey(keyDir)
			id := authenticate(ctx, first)
			Expect(keys.NewWriter(nil).Delete(ctx, id.Credential)).To(Succeed())
			Expect(openMockDriver(ctx, logger, cfg).Close()).To(Succeed())
			second := readSavedKey(keyDir)
			Expect(second).ToNot(Equal(first))
			Expect(authenticate(ctx, second).Subject).To(Equal(rackKey.OntologyID()))
		})

		It("Should delete the API key it could not save", func(ctx SpecContext) {
			logger, _ := newTestLogger()
			keyDir := GinkgoT().TempDir()
			Expect(os.Mkdir(filepath.Join(keyDir, "embedded-driver-key.tmp"), 0o700)).
				To(Succeed())
			count := func() int {
				GinkgoHelper()
				return MustSucceed(keys.NewRetrieve().
					Where(apikey.MatchSubjects(rackKey.OntologyID())).
					Count(ctx, nil))
			}
			before := count()
			Expect(driver.Open(ctx, driver.Config{
				Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
				FS:              mockFS,
				Insecure:        new(true),
				Address:         "localhost:9090",
				ParentDirname:   GinkgoT().TempDir(),
				BinaryDirname:   GinkgoT().TempDir(),
				KeyDirname:      keyDir,
				Keys:            keys,
				RackKey:         rackKey,
			})).Error().To(MatchError(syscall.EISDIR))
			Expect(count()).To(Equal(before))
		})

		It("Should keep the Driver's state inside its working directory", func(
			ctx SpecContext,
		) {
			logger, _ := newTestLogger()
			dir := GinkgoT().TempDir()
			d := openMockDriver(ctx, logger, driver.Config{ParentDirname: dir})
			Expect(filepath.Join(dir, "driver", "state.json")).To(BeAnExistingFile())
			Expect(d.Close()).To(Succeed())
		})

		It("Should reuse the extracted binary across Core processes", func(
			ctx SpecContext,
		) {
			logger, _ := newTestLogger()
			// Each open gets its own work directory, as each Core process does.
			cfg := driver.Config{BinaryDirname: GinkgoT().TempDir()}
			Expect(openMockDriver(ctx, logger, cfg).Close()).To(Succeed())
			first := MustSucceed(os.Stat(extractedBinary(cfg.BinaryDirname)))
			Expect(openMockDriver(ctx, logger, cfg).Close()).To(Succeed())
			second := MustSucceed(os.Stat(extractedBinary(cfg.BinaryDirname)))
			Expect(os.SameFile(first, second)).To(BeTrue())
		})

		It("Should replace the extracted binary when the binary changes", func(
			ctx SpecContext,
		) {
			logger, _ := newTestLogger()
			binDir := GinkgoT().TempDir()
			Expect(openMockDriver(ctx, logger, driver.Config{BinaryDirname: binDir}).
				Close()).To(Succeed())
			old := extractedBinary(binDir)
			changed := append(
				MustSucceed(fs.ReadFile(mockFS, driverName())),
				[]byte("changed")...,
			)
			Expect(openMockDriver(ctx, logger, driver.Config{
				BinaryDirname: binDir,
				FS:            fstest.MapFS{driverName(): {Data: changed}},
			}).Close()).To(Succeed())
			Expect(old).ToNot(BeAnExistingFile())
			Expect(os.ReadFile(extractedBinary(binDir))).To(Equal(changed))
		})

		It("Should remove old binaries and keep temp files", func(ctx SpecContext) {
			logger, _ := newTestLogger()
			// A '[' is pattern syntax to filepath.Glob.
			binDir := filepath.Join(GinkgoT().TempDir(), "cache[")
			Expect(os.Mkdir(binDir, 0o700)).To(Succeed())
			stale := filepath.Join(binDir, oldBinaryName())
			Expect(os.WriteFile(stale, []byte("old"), 0o700)).To(Succeed())
			temp := filepath.Join(binDir, oldBinaryName()+".123.tmp")
			Expect(os.WriteFile(temp, []byte("partial"), 0o600)).To(Succeed())
			Expect(openMockDriver(ctx, logger, driver.Config{BinaryDirname: binDir}).
				Close()).To(Succeed())
			Expect(stale).ToNot(BeAnExistingFile())
			Expect(temp).To(BeAnExistingFile())
			Expect(extractedBinary(binDir)).To(BeAnExistingFile())
		})

		DescribeTable(
			"Should start when old binaries cannot be cleaned up",
			func(ctx SpecContext, opt fstestutil.FaultyFSOption, message string) {
				logger, buffer := newTestLogger()
				binDir := GinkgoT().TempDir()
				stale := filepath.Join(binDir, oldBinaryName())
				Expect(os.WriteFile(stale, []byte("old"), 0o700)).To(Succeed())
				Expect(openMockDriver(ctx, logger, driver.Config{
					BinaryDirname: binDir,
					BinaryFS:      fstestutil.WrapFaultyFS(xfs.Default, opt),
				}).Close()).To(Succeed())
				Expect(stale).To(BeAnExistingFile())
				Expect(buffer.String()).To(ContainSubstring(message))
			},
			Entry(
				"list",
				fstestutil.WithFailList(),
				"failed to list old Driver binaries",
			),
			Entry(
				"remove",
				fstestutil.WithFailRemove(),
				"failed to remove old Driver binary",
			),
		)

		It("Should use a binary that another Core wrote first", func(
			ctx SpecContext,
		) {
			logger, _ := newTestLogger()
			binDir := GinkgoT().TempDir()
			Expect(openMockDriver(ctx, logger, driver.Config{
				BinaryDirname: binDir,
				BinaryFS:      racingFS{FS: xfs.Default},
			}).Close()).To(Succeed())
			Expect(os.ReadDir(binDir)).To(HaveLen(1))
			Expect(extractedBinary(binDir)).To(BeAnExistingFile())
		})

		It("Should write no trust anchors in insecure mode", func(ctx SpecContext) {
			logger, _ := newTestLogger()
			dir := GinkgoT().TempDir()
			envFile := filepath.Join(GinkgoT().TempDir(), "env")
			Expect(os.Setenv("MOCK_ENV_DUMP_FILE", envFile)).To(Succeed())
			defer func() { Expect(os.Unsetenv("MOCK_ENV_DUMP_FILE")).To(Succeed()) }()
			d := openMockDriver(ctx, logger, driver.Config{
				ParentDirname:   dir,
				TrustAnchorsPEM: []byte("-----BEGIN CERTIFICATE-----\nanchors\n"),
			})
			Expect(readDriverConnection(dir)).To(HaveKeyWithValue("secure", false))
			Expect(filepath.Join(dir, "driver", "trust-anchors.pem")).
				ToNot(BeAnExistingFile())
			Expect(os.ReadFile(envFile)).
				ToNot(ContainSubstring("GRPC_DEFAULT_SSL_ROOTS_FILE_PATH"))
			Expect(d.Close()).To(Succeed())
		})
	})
})

func driverName() string {
	if runtime.GOOS == "windows" {
		return "driver.exe"
	}
	return "driver"
}

// oldBinaryName returns the name of a binary that another Driver version extracted.
func oldBinaryName() string {
	return "driver-0000000000000000" + filepath.Ext(driverName())
}

// extractedBinary returns the path of the only Driver binary in binDir.
func extractedBinary(binDir string) string {
	GinkgoHelper()
	var binaries []string
	for _, e := range MustSucceed(os.ReadDir(binDir)) {
		if strings.HasPrefix(e.Name(), "driver-") &&
			!strings.HasSuffix(e.Name(), ".tmp") {
			binaries = append(binaries, filepath.Join(binDir, e.Name()))
		}
	}
	Expect(binaries).To(HaveLen(1))
	return binaries[0]
}

// authenticate returns the identity the API key logs in as.
func authenticate(ctx context.Context, key string) auth.Identity {
	GinkgoHelper()
	return MustSucceed(keys.Authenticate(ctx, auth.Proof{
		Variant: auth.APIKeyProof{APIKeyCredentials: auth.APIKeyCredentials{Key: key}},
	}))
}

// readSavedKey returns the API key the Driver saved in keyDir.
func readSavedKey(keyDir string) string {
	GinkgoHelper()
	path := filepath.Join(keyDir, "embedded-driver-key")
	return string(MustSucceed(os.ReadFile(path)))
}

// readDriverConnection returns the connection block of the config file the Driver was
// started with. It must be called while the Driver is running, since the file is
// removed once the process exits.
func readDriverConnection(parentDirname string) map[string]any {
	GinkgoHelper()
	b := MustSucceed(os.ReadFile(filepath.Join(parentDirname, "driver", "config.json")))
	var cfg map[string]any
	Expect(json.Unmarshal(b, &cfg)).To(Succeed())
	return cfg["connection"].(map[string]any)
}

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
			Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
			FS:              mockFS,
			Insecure:        new(true),
			Address:         "localhost:9090",
			Keys:            keys,
			RackKey:         rackKey,
			ParentDirname:   GinkgoT().TempDir(),
			BinaryDirname:   GinkgoT().TempDir(),
			// The Driver never starts, so Open returns as soon as the policy gives up.
			// StartTimeout only has to outlast three subprocess launches on a loaded
			// machine; it is never actually waited out.
			StartTimeout:        time.Minute,
			StopTimeout:         500 * time.Millisecond,
			RestartBaseInterval: time.Millisecond,
			RestartMaxRetries:   2,
		})
		Expect(err).To(MatchError(ContainSubstring("exceeded restart limit")))
		Expect(d).To(BeNil())
		Expect(buffer.String()).To(ContainSubstring("exceeded restart limit"))
	})

	It("Should launch the same binary after a restart", func(ctx SpecContext) {
		Expect(os.Setenv("MOCK_EXIT_AFTER_MS", "100")).To(Succeed())
		defer func() { Expect(os.Unsetenv("MOCK_EXIT_AFTER_MS")).To(Succeed()) }()
		logger, buffer := newTestLogger()
		binDir := GinkgoT().TempDir()
		d := openMockDriver(ctx, logger, driver.Config{
			BinaryDirname:       binDir,
			StartTimeout:        5 * time.Second,
			RestartBaseInterval: time.Millisecond,
		})
		first := MustSucceed(os.Stat(extractedBinary(binDir)))
		Eventually(func() int {
			return strings.Count(buffer.String(), "started successfully")
		}, 5*time.Second).Should(BeNumerically(">=", 2))
		Expect(d.Close()).To(Succeed())
		Expect(os.SameFile(first, MustSucceed(os.Stat(extractedBinary(binDir))))).
			To(BeTrue())
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
