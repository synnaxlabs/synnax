// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	"runtime"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// inUseFrom returns the live heap bytes whose allocation stack passes through a
// function whose name contains fn.
func inUseFrom(fn string) int64 {
	for range 3 {
		runtime.GC()
	}
	var records []runtime.MemProfileRecord
	n, _ := runtime.MemProfile(nil, false)
	for {
		records = make([]runtime.MemProfileRecord, n+64)
		var ok bool
		if n, ok = runtime.MemProfile(records, false); ok {
			records = records[:n]
			break
		}
	}
	var total int64
	for _, r := range records {
		frames := runtime.CallersFrames(r.Stack())
		for {
			f, more := frames.Next()
			if strings.Contains(f.Function, fn) {
				total += r.InUseBytes()
				break
			}
			if !more {
				break
			}
		}
	}
	return total
}

var _ = Describe(
	"Checker r2 mem: a big Gorp transaction followed by small ones",
	func() {
		It(
			"Should not pin the gone keys of a transaction that deletes and recreates entries holding none",
			func(ctx SpecContext) {
				prev := runtime.MemProfileRate
				runtime.MemProfileRate = 1
				DeferCleanup(func() { runtime.MemProfileRate = prev })
				db := DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
				StartRun(db.Run)
				MustOpen(
					gorp.OpenTable(
						ctx,
						gorp.TableConfig[string, r2Label]{DB: db, Name: "r2label"},
					),
				)
				labels := gorp.NewKeyReference(
					gorp.KeyReferenceConfig[int32, r2Item, string]{
						Name:     "labels",
						Table:    "r2label",
						OnDelete: gorp.Restrict,
						Kind:     gorp.Use,
						Targets:  func(e *r2Item) []string { return e.Labels },
					},
				)
				items := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[int32, r2Item]{
					DB:         db,
					Name:       "r2item",
					References: []gorp.Reference[int32, r2Item]{labels},
				}))
				const n = 50_000
				entries := make([]r2Item, n)
				keys := make([]int32, n)
				for i := range n {
					keys[i] = int32(i)
					entries[i] = r2Item{Key: int32(i), Name: "x"}
				}
				Expect(items.NewCreate().Entries(&entries).Exec(ctx, db)).To(Succeed())
				var none []r2Item
				Expect(
					items.NewRetrieve().
						Where(labels.FilterHolding(false)).
						Entries(&none).
						Exec(ctx, db),
				).To(Succeed())
				Expect(none).To(HaveLen(n))
				before := inUseFrom("gorp/internal/index.")
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					if err := items.NewDelete().
						Where(gorp.MatchKeys[int32, r2Item](keys...)).
						Exec(ctx, tx); err != nil {
						return err
					}
					return items.NewCreate().Entries(&entries).Exec(ctx, tx)
				})).To(Succeed())
				for i := range 3 {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						if err := items.NewDelete().
							Where(gorp.MatchKeys[int32, r2Item](int32(i))).
							Exec(ctx, tx); err != nil {
							return err
						}
						return items.NewCreate().Entry(&entries[i]).Exec(ctx, tx)
					})).To(Succeed())
				}
				after := inUseFrom("gorp/internal/index.")
				Expect(after - before).To(BeNumerically("<", 64<<10))
			},
		)
	},
)
