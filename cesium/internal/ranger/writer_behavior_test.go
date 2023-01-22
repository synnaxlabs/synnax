// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/ranger"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"sync"
)

var _ = Describe("WriterBehavior", func() {
	var db *ranger.DB
	BeforeEach(func() {
		db = MustSucceed(ranger.Open(ranger.Config{FS: fs.NewMem()}))
	})
	AfterEach(func() { Expect(db.Close()).To(Succeed()) })
	Describe("Start Validation", func() {
		Context("No range overlap", func() {
			It("Should successfully open the writer", func() {
				w := MustSucceed(db.NewWriter(ranger.WriterConfig{
					Start: 10 * telem.SecondTS,
				}))
				Expect(w.Close()).To(Succeed())
			})
		})
		Context("TimeRange overlap", func() {
			It("Should fail to open the writer", func() {
				w := MustSucceed(db.NewWriter(ranger.WriterConfig{
					Start: 10 * telem.SecondTS,
				}))
				Expect(w.Commit(15 * telem.SecondTS)).To(Succeed())
				Expect(w.Close()).To(Succeed())
				_, err := db.NewWriter(ranger.WriterConfig{
					Start: 10 * telem.SecondTS,
				})
				Expect(err).To(HaveOccurredAs(ranger.ErrRangeOverlap))
			})
		})
	})
	Describe("End Validation", func() {
		Context("No range overlap", func() {
			It("Should successfully commit", func() {
				w := MustSucceed(db.NewWriter(ranger.WriterConfig{
					Start: 10 * telem.SecondTS,
				}))
				MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
				Expect(w.Commit(20 * telem.SecondTS)).To(Succeed())
				Expect(w.Close()).To(Succeed())
			})
		})
		Context("TimeRange overlap", func() {
			It("Should fail to commit", func() {
				w := MustSucceed(db.NewWriter(ranger.WriterConfig{
					Start: 10 * telem.SecondTS,
				}))
				MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
				Expect(w.Commit(20 * telem.SecondTS)).To(Succeed())
				Expect(w.Close()).To(Succeed())
				w = MustSucceed(db.NewWriter(ranger.WriterConfig{
					Start: 4 * telem.SecondTS,
				}))
				MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
				Expect(w.Commit(15 * telem.SecondTS)).To(HaveOccurredAs(ranger.ErrRangeOverlap))
				Expect(w.Close()).To(Succeed())
			})
		})
		Context("Commit before start", func() {
			It("Should fail to commit", func() {
				w := MustSucceed(db.NewWriter(ranger.WriterConfig{
					Start: 10 * telem.SecondTS,
				}))
				MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
				Expect(w.Commit(5 * telem.SecondTS)).To(HaveOccurredAs(validate.Error))
				Expect(w.Close()).To(Succeed())
			})
		})
		Describe("End of one range is the start of another", func() {
			It("Should successfully commit", func() {
				w := MustSucceed(db.NewWriter(ranger.WriterConfig{
					Start: 10 * telem.SecondTS,
				}))
				MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
				Expect(w.Commit(20 * telem.SecondTS)).To(Succeed())
				Expect(w.Close()).To(Succeed())
				w = MustSucceed(db.NewWriter(ranger.WriterConfig{
					Start: 20 * telem.SecondTS,
				}))
				MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
				Expect(w.Commit(30 * telem.SecondTS)).To(Succeed())
				Expect(w.Close()).To(Succeed())
			})
		})
		Context("Multi Commit", func() {
			It("Should correctly commit a writer multiple times", func() {
				w := MustSucceed(db.NewWriter(ranger.WriterConfig{
					Start: 10 * telem.SecondTS,
				}))
				MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
				Expect(w.Commit(20 * telem.SecondTS)).To(Succeed())
				MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
				Expect(w.Commit(30 * telem.SecondTS)).To(Succeed())
				Expect(w.Close()).To(Succeed())
			})
			Context("Commit before previous commit", func() {
				It("Should fail to commit", func() {
					w := MustSucceed(db.NewWriter(ranger.WriterConfig{
						Start: 10 * telem.SecondTS,
					}))
					MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
					Expect(w.Commit(15 * telem.SecondTS)).To(Succeed())
					Expect(w.Commit(14 * telem.SecondTS)).To(HaveOccurredAs(validate.Error))
					Expect(w.Close()).To(Succeed())
				})
			})
		})
		Context("Concurrent Writes", func() {
			It("Should fail to commit one of the writes", func() {
				writerCount := 20
				errors := make([]error, writerCount)
				writers := make([]*ranger.Writer, writerCount)
				var wg sync.WaitGroup
				wg.Add(writerCount)
				for i := 0; i < writerCount; i++ {
					writers[i] = MustSucceed(db.NewWriter(ranger.WriterConfig{
						Start: 10 * telem.SecondTS,
					}))
				}
				for i, w := range writers {
					go func(i int, w *ranger.Writer) {
						defer wg.Done()
						MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
						errors[i] = w.Commit(15 * telem.SecondTS)
					}(i, w)
				}
				wg.Wait()

				occurred := lo.Filter(errors, func(err error, i int) bool {
					return err != nil
				})
				Expect(occurred).To(HaveLen(writerCount - 1))
				for _, err := range occurred {
					Expect(err).To(HaveOccurredAs(ranger.ErrRangeOverlap))
				}
			})
		})
	})
})
