// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package inspect_test

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/inspect"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

const (
	idxKey  cesium.ChannelKey = 1
	dataKey cesium.ChannelKey = 2
	strKey  cesium.ChannelKey = 3
)

// base is a plausible write time so healthy fixtures trip no time-bounds checks.
var base = telem.NewTimeStamp(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC))

var _ = Describe("Run", func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, func() {
			var fs xfs.FS
			BeforeEach(func() { fs = openFS() })

			run := func(
				ctx context.Context, overrides ...inspect.Config,
			) inspect.Report {
				GinkgoHelper()
				cfg := inspect.Config{FS: fs, Instrumentation: PanicLogger()}
				return MustSucceed(inspect.Run(
					ctx, append([]inspect.Config{cfg}, overrides...)...,
				))
			}

			findings := func(rep inspect.Report, check inspect.Check) []inspect.Finding {
				var matched []inspect.Finding
				for _, f := range rep.Findings {
					if f.Check == check {
						matched = append(matched, f)
					}
				}
				for _, ch := range rep.Channels {
					for _, f := range ch.Findings {
						if f.Check == check {
							matched = append(matched, f)
						}
					}
				}
				return matched
			}

			channelReport := func(
				rep inspect.Report, key cesium.ChannelKey,
			) inspect.ChannelReport {
				GinkgoHelper()
				for _, ch := range rep.Channels {
					if ch.Key == key {
						return ch
					}
				}
				Fail("no report for channel")
				return inspect.ChannelReport{}
			}

			openDB := func(ctx context.Context) *cesium.DB {
				GinkgoHelper()
				return MustSucceed(cesium.Open(
					ctx, "",
					cesium.WithFS(fs),
					cesium.WithInstrumentation(PanicLogger()),
				))
			}

			createIndexed := func(ctx context.Context, db *cesium.DB) {
				GinkgoHelper()
				Expect(db.CreateChannel(
					ctx,
					cesium.Channel{
						Key:      idxKey,
						Name:     "time",
						IsIndex:  true,
						DataType: telem.TimestampT,
					},
					cesium.Channel{
						Key:      dataKey,
						Name:     "values",
						Index:    idxKey,
						DataType: telem.Int64T,
					},
				)).To(Succeed())
			}

			writeAt := func(
				ctx context.Context,
				db *cesium.DB,
				start telem.TimeStamp,
				n int,
				spacing telem.TimeSpan,
			) {
				GinkgoHelper()
				ts := make([]telem.TimeStamp, n)
				data := make([]int64, n)
				for i := range n {
					ts[i] = start.Add(telem.TimeSpan(i) * spacing)
					data[i] = int64(i)
				}
				Expect(db.Write(ctx, start, telem.MultiFrame(
					[]cesium.ChannelKey{idxKey, dataKey},
					[]telem.Series{telem.NewSeries(ts), telem.NewSeries(data)},
				))).To(Succeed())
			}

			// createHealthy builds two channels holding three domains of three
			// samples each, spaced far apart.
			createHealthy := func(ctx context.Context) {
				GinkgoHelper()
				db := openDB(ctx)
				createIndexed(ctx, db)
				for i := range 3 {
					writeAt(
						ctx, db,
						base.Add(telem.TimeSpan(i*100)*telem.Second),
						3, telem.Second,
					)
				}
				Expect(db.Close()).To(Succeed())
			}

			patch := func(path string, off int64, b []byte) {
				GinkgoHelper()
				f := MustSucceed(fs.Open(path, os.O_RDWR))
				defer func() { Expect(f.Close()).To(Succeed()) }()
				MustSucceed(f.WriteAt(b, off))
			}

			createFile := func(path string, contents []byte) {
				GinkgoHelper()
				f := MustSucceed(fs.Open(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC))
				defer func() { Expect(f.Close()).To(Succeed()) }()
				if len(contents) > 0 {
					MustSucceed(f.Write(contents))
				}
			}

			appendTo := func(path string, b []byte) {
				GinkgoHelper()
				f := MustSucceed(fs.Open(path, os.O_RDWR))
				defer func() { Expect(f.Close()).To(Succeed()) }()
				info := MustSucceed(f.Stat())
				MustSucceed(f.WriteAt(b, info.Size()))
			}

			Describe("Healthy database", func() {
				It("Should report clean statistics and no findings", func(
					ctx SpecContext,
				) {
					createHealthy(ctx)
					rep := run(ctx)
					Expect(rep.Totals.Channels).To(Equal(2))
					Expect(rep.Totals.Errors).To(BeZero())
					Expect(rep.Totals.Warnings).To(BeZero())
					Expect(rep.Totals.Infos).To(BeZero())
					Expect(rep.Totals.Domains).To(Equal(6))
					Expect(rep.Totals.Samples).To(Equal(int64(18)))
					Expect(rep.Findings).To(BeEmpty())

					idx := channelReport(rep, idxKey)
					Expect(idx.Channel.IsIndex).To(BeTrue())
					Expect(idx.Stats.Domains).To(Equal(3))
					Expect(idx.Stats.Files).To(Equal(1))
					Expect(idx.Stats.Samples).To(Equal(int64(9)))
					Expect(idx.Stats.SamplesExact).To(BeTrue())
					Expect(idx.Stats.LiveBytes).To(Equal(72 * telem.Byte))
					Expect(idx.Stats.GarbageBytes).To(BeZero())
					Expect(idx.Stats.Gaps).To(Equal(2))
					Expect(idx.Stats.MicroGaps).To(BeZero())
					Expect(idx.Stats.TinyDomains).To(BeZero())
					Expect(idx.Stats.TimeRange.Start).To(Equal(base))
					Expect(idx.Stats.DomainSizes.Min).To(Equal(24 * telem.Byte))
					Expect(idx.Stats.DomainSizes.Max).To(Equal(24 * telem.Byte))
					Expect(idx.Stats.DiskBytes).To(BeNumerically(">", 0))

					data := channelReport(rep, dataKey)
					Expect(data.Channel.Index).To(Equal(idxKey))
					Expect(data.Stats.Domains).To(Equal(3))
					Expect(data.Stats.Samples).To(Equal(int64(9)))
				})
			})

			Describe("Garbage", func() {
				It("Should report bytes stranded by a delete", func(ctx SpecContext) {
					db := openDB(ctx)
					createIndexed(ctx, db)
					writeAt(ctx, db, base, 3, telem.Second)
					Expect(db.DeleteTimeRange(
						ctx,
						[]cesium.ChannelKey{dataKey},
						base.Add(telem.Second).Range(base.Add(2*telem.Second)),
					)).To(Succeed())
					Expect(db.Close()).To(Succeed())
					rep := run(ctx)
					data := channelReport(rep, dataKey)
					Expect(data.Stats.GarbageBytes).To(Equal(8 * telem.Byte))
					Expect(data.Stats.Domains).To(Equal(2))
					Expect(channelReport(rep, idxKey).Stats.GarbageBytes).To(BeZero())
				})

				It("Should warn when garbage dominates the channel", func(
					ctx SpecContext,
				) {
					const samples = 150_000
					db := openDB(ctx)
					createIndexed(ctx, db)
					writeAt(ctx, db, base, samples, telem.Second)
					Expect(db.DeleteTimeRange(
						ctx,
						[]cesium.ChannelKey{dataKey},
						base.Add(telem.Second).Range(
							base.Add((samples-1)*telem.Second),
						),
					)).To(Succeed())
					Expect(db.Close()).To(Succeed())
					rep := run(ctx, inspect.Config{FileSize: telem.Megabyte})
					data := channelReport(rep, dataKey)
					Expect(data.Stats.GarbageRatio).To(BeNumerically(">", 0.5))
					Expect(data.Stats.GCEligibleFiles).To(BeNumerically(">=", 1))
					Expect(findings(rep, inspect.CheckGarbage)).To(HaveLen(1))
				})
			})

			Describe("Variable-length channels", func() {
				createStrings := func(ctx context.Context) {
					GinkgoHelper()
					db := openDB(ctx)
					createIndexed(ctx, db)
					Expect(db.CreateChannel(ctx, cesium.Channel{
						Key:      strKey,
						Name:     "labels",
						Index:    idxKey,
						DataType: telem.StringT,
					})).To(Succeed())
					ts := []telem.TimeStamp{
						base,
						base.Add(telem.Second),
						base.Add(2 * telem.Second),
					}
					Expect(db.Write(ctx, base, telem.MultiFrame(
						[]cesium.ChannelKey{idxKey, strKey},
						[]telem.Series{
							telem.NewSeries(ts),
							telem.NewSeriesV("aa", "bbb", "c"),
						},
					))).To(Succeed())
					Expect(db.Close()).To(Succeed())
				}

				It("Should count samples with a deep walk", func(ctx SpecContext) {
					createStrings(ctx)
					rep := run(ctx)
					str := channelReport(rep, strKey)
					Expect(str.Stats.Samples).To(Equal(int64(3)))
					Expect(str.Stats.SamplesExact).To(BeTrue())
					Expect(findings(rep, inspect.CheckVarLenWalk)).To(BeEmpty())
				})

				It("Should leave samples inexact when deep checks are off", func(
					ctx SpecContext,
				) {
					createStrings(ctx)
					rep := run(ctx, inspect.Config{Deep: new(false)})
					str := channelReport(rep, strKey)
					Expect(str.Stats.SamplesExact).To(BeFalse())
				})

				It("Should detect corrupt sample framing", func(ctx SpecContext) {
					createStrings(ctx)
					// Inflate the first length prefix far past the domain's end.
					patch("3/1.domain", 0, []byte{0xFF, 0xFF, 0xFF, 0x0F})
					rep := run(ctx)
					Expect(findings(rep, inspect.CheckVarLenWalk)).To(HaveLen(1))
					Expect(channelReport(rep, strKey).Stats.SamplesExact).To(BeFalse())
				})
			})

			Describe("Root findings", func() {
				It("Should flag stray files and directories", func(ctx SpecContext) {
					createHealthy(ctx)
					createFile("stray.txt", []byte("junk"))
					sub := MustSucceed(fs.Sub("junkdir"))
					f := MustSucceed(sub.Open("x", os.O_CREATE|os.O_WRONLY))
					Expect(f.Close()).To(Succeed())
					orphans := findings(run(ctx), inspect.CheckOrphanFile)
					subjects := make([]string, len(orphans))
					for i, o := range orphans {
						subjects[i] = o.Subject
					}
					Expect(subjects).To(ConsistOf("stray.txt", "junkdir"))
				})

				It("Should flag directories from an interrupted delete", func(
					ctx SpecContext,
				) {
					createHealthy(ctx)
					sub := MustSucceed(fs.Sub("5-DELETE-123"))
					f := MustSucceed(sub.Open("meta.json", os.O_CREATE|os.O_WRONLY))
					Expect(f.Close()).To(Succeed())
					artifacts := findings(run(ctx), inspect.CheckArtifacts)
					Expect(artifacts).To(HaveLen(1))
					Expect(artifacts[0].Subject).To(Equal("5-DELETE-123"))
				})
			})

			Describe("Index integrity", func() {
				It("Should flag a trailing partial record", func(ctx SpecContext) {
					createHealthy(ctx)
					appendTo("2/index.domain", []byte{1, 2, 3})
					rep := run(ctx)
					decode := findings(rep, inspect.CheckIndexDecode)
					Expect(decode).To(HaveLen(1))
					Expect(decode[0].Channel).To(Equal(dataKey))
					Expect(channelReport(rep, dataKey).Stats.Domains).To(Equal(3))
				})

				It("Should flag an unreadable index with data present", func(
					ctx SpecContext,
				) {
					createHealthy(ctx)
					Expect(fs.Remove("2/index.domain")).To(Succeed())
					rep := run(ctx)
					decode := findings(rep, inspect.CheckIndexDecode)
					Expect(decode).To(HaveLen(1))
					Expect(decode[0].Message).To(ContainSubstring("unreadable"))
				})

				It("Should flag out-of-order and overlapping domains", func(
					ctx SpecContext,
				) {
					createHealthy(ctx)
					f := MustSucceed(fs.Open("2/index.domain", os.O_RDWR))
					b := make([]byte, 52)
					MustSucceed(f.ReadAt(b, 0))
					MustSucceed(f.WriteAt(b[26:52], 0))
					MustSucceed(f.WriteAt(b[:26], 26))
					Expect(f.Close()).To(Succeed())
					rep := run(ctx)
					Expect(findings(rep, inspect.CheckDomainOrder)).To(HaveLen(1))
					Expect(findings(rep, inspect.CheckDomainOverlap)).To(HaveLen(1))
				})

				It("Should flag a zero file key", func(ctx SpecContext) {
					createHealthy(ctx)
					patch("2/index.domain", 16, []byte{0, 0})
					Expect(findings(run(ctx), inspect.CheckFileKey)).To(HaveLen(1))
				})

				It("Should flag file keys above the counter", func(ctx SpecContext) {
					createHealthy(ctx)
					patch("2/counter.domain", 0, []byte{0, 0, 0, 0})
					fkey := findings(run(ctx), inspect.CheckFileKey)
					Expect(fkey).To(HaveLen(1))
					Expect(fkey[0].Message).To(ContainSubstring("3 of 3 domains"))
				})

				It("Should flag domains extending past their data file", func(
					ctx SpecContext,
				) {
					createHealthy(ctx)
					// First record's size field: 24 -> 96, past the 72-byte file.
					patch("2/index.domain", 22, []byte{96, 0, 0, 0})
					Expect(findings(run(ctx), inspect.CheckFileBounds)).To(HaveLen(1))
				})

				It("Should flag a missing data file", func(ctx SpecContext) {
					createHealthy(ctx)
					Expect(fs.Remove("2/1.domain")).To(Succeed())
					missing := findings(run(ctx), inspect.CheckMissingFile)
					Expect(missing).To(HaveLen(1))
					Expect(missing[0].Subject).To(Equal("1.domain"))
					Expect(missing[0].Message).To(ContainSubstring("3 domains"))
				})

				It("Should flag partial trailing samples", func(ctx SpecContext) {
					createHealthy(ctx)
					// First record's size field: 24 -> 21, no longer a multiple of 8.
					patch("2/index.domain", 22, []byte{21, 0, 0, 0})
					Expect(findings(run(ctx), inspect.CheckDensityAlign)).To(HaveLen(1))
				})
			})

			Describe("Metadata", func() {
				It("Should flag an undecodable meta.json", func(ctx SpecContext) {
					createHealthy(ctx)
					createFile("2/meta.json", []byte("{"))
					meta := findings(run(ctx), inspect.CheckMeta)
					Expect(meta).To(HaveLen(1))
					Expect(meta[0].Channel).To(Equal(dataKey))
				})

				It("Should flag a data channel whose index is gone", func(
					ctx SpecContext,
				) {
					createHealthy(ctx)
					Expect(fs.Remove("1")).To(Succeed())
					ref := findings(run(ctx), inspect.CheckIndexRef)
					Expect(ref).To(HaveLen(1))
					Expect(ref[0].Message).To(ContainSubstring("no directory"))
				})

				It("Should flag an index reference to a non-index channel", func(
					ctx SpecContext,
				) {
					createHealthy(ctx)
					MustSucceed(fs.Sub("4"))
					createFile(
						"4/meta.json",
						[]byte(`{"name":"x","data_type":"int64","key":4,`+
							`"index":2,"version":2}`),
					)
					ref := findings(run(ctx), inspect.CheckIndexRef)
					Expect(ref).To(HaveLen(1))
					Expect(ref[0].Message).To(
						ContainSubstring("not an index channel"),
					)
				})

				It("Should flag channels the engine ignores on open", func(
					ctx SpecContext,
				) {
					createHealthy(ctx)
					MustSucceed(fs.Sub("99"))
					createFile(
						"99/meta.json",
						[]byte(`{"name":"legacy","data_type":"int64","key":99,`+
							`"index":0,"version":1}`),
					)
					Expect(findings(run(ctx), inspect.CheckIgnored)).To(HaveLen(1))
				})
			})

			Describe("Channel directory contents", func() {
				It("Should flag GC and delete artifacts", func(ctx SpecContext) {
					createHealthy(ctx)
					createFile("2/1.domain_gc", []byte{1})
					createFile("2/1.domain_temp", []byte{1})
					createFile("2/meta.json.tmp", []byte{1})
					Expect(findings(run(ctx), inspect.CheckArtifacts)).To(HaveLen(3))
				})

				It("Should flag unreferenced and unrecognized files", func(
					ctx SpecContext,
				) {
					createHealthy(ctx)
					createFile("2/7.domain", []byte{1, 2, 3, 4})
					createFile("2/weird.txt", []byte("junk"))
					orphans := findings(run(ctx), inspect.CheckOrphanFile)
					subjects := make([]string, len(orphans))
					for i, o := range orphans {
						subjects[i] = o.Subject
					}
					Expect(subjects).To(ConsistOf("7.domain", "weird.txt"))
				})

				It("Should flag data files inside a virtual channel", func(
					ctx SpecContext,
				) {
					db := openDB(ctx)
					Expect(db.CreateChannel(ctx, cesium.Channel{
						Key:      9,
						Name:     "virtual",
						DataType: telem.Int64T,
						Virtual:  true,
					})).To(Succeed())
					Expect(db.Close()).To(Succeed())
					rep := run(ctx)
					virtual := channelReport(rep, 9)
					Expect(virtual.Channel.Virtual).To(BeTrue())
					Expect(virtual.Stats.Domains).To(BeZero())
					Expect(virtual.Findings).To(BeEmpty())

					createFile("9/1.domain", []byte{1, 2, 3})
					orphans := findings(run(ctx), inspect.CheckOrphanFile)
					Expect(orphans).To(HaveLen(1))
					Expect(orphans[0].Message).To(ContainSubstring("virtual"))
				})
			})

			Describe("Timestamp plausibility", func() {
				writeSingle := func(ctx context.Context, start telem.TimeStamp) {
					GinkgoHelper()
					db := openDB(ctx)
					createIndexed(ctx, db)
					writeAt(ctx, db, start, 3, telem.Second)
					Expect(db.Close()).To(Succeed())
				}

				It("Should flag near-epoch timestamps", func(ctx SpecContext) {
					writeSingle(ctx, 10*telem.SecondTS)
					bounds := findings(run(ctx), inspect.CheckTimeBounds)
					Expect(bounds).To(HaveLen(2))
					Expect(bounds[0].Message).To(
						ContainSubstring("elapsed-time-as-timestamp"),
					)
				})

				It("Should flag far-past timestamps", func(ctx SpecContext) {
					writeSingle(ctx, telem.NewTimeStamp(
						time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC),
					))
					bounds := findings(run(ctx), inspect.CheckTimeBounds)
					Expect(bounds).To(HaveLen(2))
					Expect(bounds[0].Message).To(ContainSubstring("start before"))
				})

				It("Should flag far-future timestamps", func(ctx SpecContext) {
					writeSingle(ctx, telem.Now().Add(3*24*telem.Hour))
					bounds := findings(run(ctx), inspect.CheckTimeBounds)
					Expect(bounds).To(HaveLen(2))
					Expect(bounds[0].Message).To(
						ContainSubstring("after the current time"),
					)
				})
			})

			Describe("Domain shape", func() {
				It("Should flag micro-gaps between domains", func(ctx SpecContext) {
					db := openDB(ctx)
					createIndexed(ctx, db)
					writeAt(ctx, db, base, 3, telem.Second)
					// One microsecond past the first domain's exclusive end.
					writeAt(
						ctx, db,
						base.Add(2*telem.Second+telem.Nanosecond+telem.Microsecond),
						3, telem.Second,
					)
					Expect(db.Close()).To(Succeed())
					rep := run(ctx)
					Expect(findings(rep, inspect.CheckMicroGap)).To(HaveLen(2))
					Expect(channelReport(rep, dataKey).Stats.MicroGaps).To(Equal(1))
				})

				It("Should flag channels dominated by tiny domains", func(
					ctx SpecContext,
				) {
					db := openDB(ctx)
					createIndexed(ctx, db)
					for i := range 10 {
						writeAt(
							ctx, db,
							base.Add(telem.TimeSpan(i*100)*telem.Second),
							1, telem.Second,
						)
					}
					Expect(db.Close()).To(Succeed())
					rep := run(ctx)
					Expect(findings(rep, inspect.CheckTinyDomain)).To(HaveLen(2))
					Expect(
						channelReport(rep, dataKey).Stats.TinyDomains,
					).To(Equal(10))
				})
			})

			Describe("Deep index content", func() {
				It("Should flag timestamps outside their domain bounds", func(
					ctx SpecContext,
				) {
					createHealthy(ctx)
					out := make([]byte, 8)
					telem.ByteOrder.PutUint64(
						out, uint64(base.Add(1000*24*telem.Hour)),
					)
					patch("1/1.domain", 8, out)
					content := findings(run(ctx), inspect.CheckIndexContent)
					Expect(content).To(HaveLen(1))
					Expect(content[0].Channel).To(Equal(idxKey))
				})

				It("Should flag non-increasing timestamps", func(ctx SpecContext) {
					createHealthy(ctx)
					first := make([]byte, 8)
					telem.ByteOrder.PutUint64(first, uint64(base))
					patch("1/1.domain", 8, first)
					Expect(
						findings(run(ctx), inspect.CheckIndexContent),
					).To(HaveLen(1))
				})

				It("Should skip deep checks when disabled", func(ctx SpecContext) {
					createHealthy(ctx)
					first := make([]byte, 8)
					telem.ByteOrder.PutUint64(first, uint64(base))
					patch("1/1.domain", 8, first)
					rep := run(ctx, inspect.Config{Deep: new(false)})
					Expect(findings(rep, inspect.CheckIndexContent)).To(BeEmpty())
				})
			})

			Describe("Filtering and progress", func() {
				It("Should restrict full inspection to requested channels", func(
					ctx SpecContext,
				) {
					createHealthy(ctx)
					rep := run(ctx, inspect.Config{
						Channels: []cesium.ChannelKey{dataKey},
					})
					Expect(rep.Channels).To(HaveLen(1))
					Expect(rep.Channels[0].Key).To(Equal(dataKey))
					Expect(findings(rep, inspect.CheckIndexRef)).To(BeEmpty())
				})

				It("Should report progress per channel", func(ctx SpecContext) {
					createHealthy(ctx)
					type call struct{ done, total int }
					var calls []call
					run(ctx, inspect.Config{
						Progress: func(done, total int) {
							calls = append(calls, call{done, total})
						},
					})
					Expect(calls).To(Equal([]call{{1, 2}, {2, 2}}))
				})
			})

			Describe("Configuration", func() {
				It("Should reject a missing file system", func(ctx SpecContext) {
					Expect(inspect.Run(ctx)).Error().To(
						MatchError(ContainSubstring("fs: must be non-nil")),
					)
				})
			})
		})
	}
})
