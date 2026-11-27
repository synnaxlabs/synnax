// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package storage_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/cockroachdb/pebble/v2/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/confluence"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/io/fs/testutil/pebblefs"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

type entry struct {
	Key  int32
	Name string
}

func (e entry) GorpKey() int32 { return e.Key }

// syncFailFS fails each SyncDir of dir with EIO.
type syncFailFS struct {
	xfs.FS
	dir string
}

func (f syncFailFS) SyncDir(name string) error {
	if name == f.dir {
		return syscall.EIO
	}
	return f.FS.SyncDir(name)
}

var _ = Describe("storage", func() {
	It("Should open the stores on the file systems that the config injects", func(
		ctx SpecContext,
	) {
		kvFS, tsFS := vfs.NewMem(), xfs.NewMem()
		l := MustOpen(storage.OpenLayer(ctx, storage.LayerConfig{
			Dirname: "data",
			Version: "0.0.0",
			KVFS:    kvFS,
			TSFS:    tsFS,
		}))
		StartRun(l.Run)
		Expect(l.TS.CreateChannel(ctx, cesium.Channel{
			Key: 1, Name: "time", IsIndex: true, DataType: telem.TimestampT,
		})).To(Succeed())
		Expect(kvFS.Stat(filepath.Join("data", "LOCK"))).Error().ToNot(HaveOccurred())
		Expect(MustSucceed(kvFS.List(filepath.Join("data", "kv")))).ToNot(BeEmpty())
		Expect(tsFS.Exists(filepath.Join("data", "cesium", "1"))).To(BeTrue())
	})

	DescribeTable("Should keep a committed write through a crash", func(
		ctx SpecContext,
		existing string,
	) {
		disk := MustSucceed(fault.New(fault.Config{}))
		if existing != "" {
			// An open that failed, or another process, made it with no sync.
			MustSucceed(disk.Sub(existing))
		}
		open := func(fs *fault.FS) *storage.Layer {
			GinkgoHelper()
			l := MustOpen(storage.OpenLayer(ctx, storage.LayerConfig{
				Dirname: filepath.Join("parent", "data"),
				Version: "0.0.0",
				KVFS:    pebblefs.New(fs),
				TSFS:    fs,
			}))
			StartRun(l.Run)
			return l
		}
		Expect(open(disk).KV.Set(ctx, []byte("key"), []byte("value"))).To(Succeed())
		image := MustSucceed(disk.Crash(fault.CrashConfig{}))
		Expect(image.Exists(filepath.Join("parent", "data", "cesium"))).To(BeTrue())
		value, closer := MustSucceed2(open(image).KV.Get([]byte("key")))
		Expect(value).To(Equal([]byte("value")))
		Expect(closer.Close()).To(Succeed())
	},
		Entry("in a new storage directory", ""),
		Entry(
			"in a storage directory that exists with no durable entry",
			filepath.Join("parent", "data"),
		),
		Entry(
			"in a storage directory whose parent exists with no durable entry",
			"parent",
		),
	)

	It("Should open and log a warning when a sync of the storage directory fails", func(
		ctx SpecContext,
	) {
		core, logs := observer.New(zapcore.WarnLevel)
		logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
			ZapLogger: zap.New(core),
		}))
		MustOpen(storage.OpenLayer(ctx, storage.LayerConfig{
			Instrumentation: alamos.New("storage", alamos.WithLogger(logger)),
			Dirname:         "data",
			Version:         "0.0.0",
			KVFS:            vfs.NewMem(),
			TSFS:            syncFailFS{FS: xfs.NewMem(), dir: "data"},
		}))
		entries := logs.FilterMessage("failed to sync storage directory").All()
		Expect(entries).To(HaveLen(1))
		fields := entries[0].ContextMap()
		Expect(fields).To(HaveKeyWithValue("dir", "data"))
		Expect(fields).To(HaveKeyWithValue("error", syscall.EIO.Error()))
	})

	Describe("Run", func() {
		It("Should run the time-series engine until its context ends", func(
			ctx SpecContext,
		) {
			l := MustOpen(storage.OpenLayer(ctx, storage.LayerConfig{
				InMemory: new(true),
			}))
			StartRun(l.Run)
			streamer := MustSucceed(l.TS.NewStreamer(cesium.StreamerConfig{
				SendOpenAck: true,
			}))
			requests := confluence.NewStream[cesium.StreamerRequest]()
			responses := confluence.NewStream[cesium.StreamerResponse](1)
			streamer.InFrom(requests)
			streamer.OutTo(responses)
			sCtx, cancel := signal.WithCancel(ctx)
			DeferCleanup(cancel)
			streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			// The streamer acknowledges its open only once the relay runs.
			Eventually(responses.Outlet()).Should(Receive())
			requests.Close()
			Eventually(sCtx.Stopped()).Should(BeClosed())
		})

		It("Should run the Gorp DB until its context ends", func(ctx SpecContext) {
			l := MustOpen(storage.OpenLayer(ctx, storage.LayerConfig{
				InMemory: new(true),
			}))
			StartRun(l.Run)
			// A table with indexes opens only once Run runs the Gorp DB.
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[int32, entry]{
				DB: l.Gorp,
				Indexes: []gorp.Index[int32, entry]{
					gorp.NewLookupIndex(
						"name",
						func(e *entry) string { return e.Name },
					),
				},
			}))
		})

		It("Should stop the time-series engine when the Gorp DB fails", func(
			ctx SpecContext,
		) {
			l := MustOpen(storage.OpenLayer(ctx, storage.LayerConfig{
				InMemory: new(true),
			}))
			StartRun(l.Gorp.Run)
			Expect(l.Run(ctx)).To(MatchError(ContainSubstring("run called twice")))
		})

		It("Should stop the Gorp DB when the time-series engine fails", func(
			ctx SpecContext,
		) {
			l := MustOpen(storage.OpenLayer(ctx, storage.LayerConfig{
				InMemory: new(true),
			}))
			StartRun(l.TS.Run)
			Expect(l.Run(ctx)).To(MatchError(ContainSubstring("run called twice")))
			Expect(gorp.OpenTable(ctx, gorp.TableConfig[int32, entry]{
				DB: l.Gorp,
				Indexes: []gorp.Index[int32, entry]{
					gorp.NewLookupIndex(
						"name",
						func(e *entry) string { return e.Name },
					),
				},
			})).Error().To(MatchError(ContainSubstring("after DB.Run stopped")))
		})
	})

	Describe("Open", func() {
		var (
			tempDir string
			cfg     storage.LayerConfig
		)
		BeforeEach(func() {
			tempDir = MustSucceed(os.MkdirTemp("", "synnax-test"))
			// DeferCleanup runs after the layer's Close, whereas an AfterEach would
			// fire while files are still open and fail RemoveAll on Windows.
			DeferCleanup(func() { Expect(os.RemoveAll(tempDir)).To(Succeed()) })
			cfg = storage.LayerConfig{
				Dirname: filepath.Join(tempDir, "storage"),
				Version: "0.0.0",
			}
		})
		Describe("Acquiring a lock", func() {
			It(
				"Should return an error if the lock is already acquired",
				func(ctx SpecContext) {
					MustOpen(storage.OpenLayer(ctx, cfg))
					Expect(
						storage.OpenLayer(ctx, cfg),
					).Error().
						To(MatchError(ContainSubstring("Failed to acquire lock on storage directory")))
				},
			)
		})
		Describe("Permissions", func() {
			// These two tests are failing on Windows because VFS cannot create a
			// directory on Windows with custom permissions – all directories are
			// created with 777.
			if !strings.HasPrefix(runtime.GOOS, "windows") {
				Describe("Name Directory", func() {
					It(
						"Should set the correct permissions on the storage directory",
						func(ctx SpecContext) {
							cfg.Perm = xfs.UserRWX
							MustOpen(storage.OpenLayer(ctx, cfg))
							stat := MustSucceed(os.Stat(cfg.Dirname))
							Expect(stat.Mode().Perm()).To(Equal(cfg.Perm))
						},
					)
				})
				Describe("Existing Directory", func() {
					var p string
					BeforeEach(func() {
						p = filepath.Join(tempDir, "x-storage")
						Expect(os.Mkdir(p, xfs.UserR)).To(Succeed())
						cfg.Dirname = p
					})
					AfterEach(func() { Expect(os.RemoveAll(p)).To(Succeed()) })
					It(
						"Should return an error if the directory exists but has insufficient permissions",
						func(ctx SpecContext) {
							// use os.Stat to check the dir permissions
							cfg.Perm = xfs.UserRWX
							Expect(
								storage.OpenLayer(ctx, cfg),
							).Error().
								To(MatchError(ContainSubstring(
									"requires the storage directory to have at least",
								)))
						},
					)
				})
			}
		})
		Describe("In-Memory", func() {
			It("Should open a memory backed version of storage", func(ctx SpecContext) {
				cfg.InMemory = new(true)
				MustOpen(storage.OpenLayer(ctx, cfg))
			})
		})
		Describe("Aspen keys", func() {
			It(
				"Should delete only the digests and the version counter",
				func(ctx SpecContext) {
					deleted := []string{"--dig/a", "--dig/b", "ver"}
					kept := []string{"aspen.cluster", "--dig", "--dig0", "ve", "vera"}
					l := MustSucceed(storage.OpenLayer(ctx, cfg))
					for _, key := range append(deleted, kept...) {
						Expect(
							l.KV.Set(ctx, []byte(key), []byte("value")),
						).To(Succeed())
					}
					Expect(l.Close()).To(Succeed())
					l = MustOpen(storage.OpenLayer(ctx, cfg))
					for _, key := range deleted {
						Expect(l.KV.Get([]byte(key))).Error().
							To(MatchError(query.ErrNotFound))
					}
					for _, key := range kept {
						value, closer := MustSucceed2(l.KV.Get([]byte(key)))
						Expect(value).To(Equal([]byte("value")))
						Expect(closer.Close()).To(Succeed())
					}
				},
			)
		})
	})
	Describe("ServiceConfig", func() {
		DescribeTable("Validate", func(
			spec func(storage.LayerConfig) storage.LayerConfig,
			contains string,
		) {
			iCfg := storage.LayerConfig{
				Perm:                 xfs.UserRWX,
				InMemory:             new(false),
				KVEngine:             storage.KVEnginePebble,
				TSEngine:             storage.TSEngineCesium,
				UpdateBackupDisabled: new(false),
			}
			iCfg.Dirname = "foo"
			iCfg.Version = "0.60.0"
			err := spec(iCfg).Validate()
			if contains == "" {
				Expect(err).ToNot(HaveOccurred())
			} else {
				Expect(err).To(MatchError(ContainSubstring(contains)))
			}
		},
			Entry("Directory not set",
				func(cfg storage.LayerConfig) storage.LayerConfig {
					cfg.Dirname = ""
					cfg.InMemory = new(false)
					return cfg
				},
				"dirname",
			),
			Entry("Directory not set, mem-backed",
				func(cfg storage.LayerConfig) storage.LayerConfig {
					cfg.Dirname = ""
					cfg.InMemory = new(true)
					return cfg
				},
				"",
			),
			Entry("Invalid key-value engine",
				func(cfg storage.LayerConfig) storage.LayerConfig {
					cfg.KVEngine = 12
					return cfg
				},
				"key-value engine",
			),
			Entry("Invalid permissions",
				func(cfg storage.LayerConfig) storage.LayerConfig {
					cfg.Perm = 0
					return cfg
				},
				"permissions",
			),
			Entry("Valid config", func(cfg storage.LayerConfig) storage.LayerConfig {
				return cfg
			}, ""),
			Entry("Version not set",
				func(cfg storage.LayerConfig) storage.LayerConfig {
					cfg.Version = ""
					return cfg
				},
				"version",
			),
			Entry("Version with a v prefix",
				func(cfg storage.LayerConfig) storage.LayerConfig {
					cfg.Version = "v0.60.0"
					return cfg
				},
				"version",
			),
			Entry("Version not set, mem-backed",
				func(cfg storage.LayerConfig) storage.LayerConfig {
					cfg.Version = ""
					cfg.InMemory = new(true)
					return cfg
				},
				"",
			),
		)
	})
})
