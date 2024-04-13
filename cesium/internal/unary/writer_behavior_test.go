// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unary_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Writer Behavior", func() {
	Describe("Index", func() {
		var db *unary.DB
		BeforeEach(func() {
			db = MustSucceed(unary.Open(unary.Config{
				FS: fs.NewMem(),
				Channel: core.Channel{
					Key:      2,
					DataType: telem.TimeStampT,
					IsIndex:  true,
				},
			}))
		})
		AfterEach(func() {
			Expect(db.Close()).To(Succeed())
		})
		Specify("Happy Path", func() {
			w, t := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
				Start:   telem.TimeStamp(0),
				Subject: control.Subject{Key: "foo"},
			}))
			Expect(t.Occurred()).To(BeTrue())
			Expect(MustSucceed(w.Write(telem.NewSecondsTSV(0, 1, 2, 3, 4, 5)))).To(Equal(telem.Alignment(0)))
			Expect(MustSucceed(w.Write(telem.NewSecondsTSV(6, 7, 8, 9, 10, 11)))).To(Equal(telem.Alignment(6)))
			Expect(MustSucceed(w.Commit(ctx))).To(Equal(11*telem.SecondTS + 1))
			t = MustSucceed(w.Close())
			Expect(t.Occurred()).To(BeTrue())
			Expect(db.LeadingControlState()).To(BeNil())
		})
	})
	Describe("Channel Indexed", func() {
		var (
			db      *unary.DB
			indexDB *unary.DB
			index   uint32 = 1
			data    uint32 = 2
		)
		BeforeEach(func() {
			indexDB = MustSucceed(unary.Open(unary.Config{
				FS: fs.NewMem(),
				Channel: core.Channel{
					Key:      index,
					DataType: telem.TimeStampT,
					IsIndex:  true,
				},
			}))
			db = MustSucceed(unary.Open(unary.Config{
				FS: fs.NewMem(),
				Channel: core.Channel{
					Key:      data,
					DataType: telem.Int64T,
					Index:    index,
				},
			}))
			db.SetIndex(indexDB.Index())
		})
		AfterEach(func() {
			Expect(db.Close()).To(Succeed())
			Expect(indexDB.Close()).To(Succeed())
		})
		Specify("Happy Path", func() {
			Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20))).To(Succeed())
			w, t := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
				Start:   10 * telem.SecondTS,
				Subject: control.Subject{Key: "foo"},
			}))
			By("Taking control of the DB")
			Expect(db.LeadingControlState().Subject).To(Equal(control.Subject{Key: "foo"}))
			Expect(t.Occurred()).To(BeTrue())
			Expect(MustSucceed(w.Write(telem.NewSeries([]int64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10})))).To(Equal(telem.Alignment(0)))
			Expect(MustSucceed(w.Commit(ctx))).To(Equal(20*telem.SecondTS + 1))
			t = MustSucceed(w.Close())
			Expect(t.Occurred()).To(BeTrue())
			By("Releasing control of the DB")
			Expect(db.LeadingControlState()).To(BeNil())
		})
	})
	Describe("Control", func() {
		Describe("Index", func() {
			var db *unary.DB
			BeforeEach(func() {
				db = MustSucceed(unary.Open(unary.Config{
					FS: fs.NewMem(),
					Channel: core.Channel{
						Key:      2,
						DataType: telem.TimeStampT,
						IsIndex:  true,
					},
				}))
			})
			AfterEach(func() {
				Expect(db.Close()).To(Succeed())
			})
			Specify("Control Handoff", func() {
				w1, t := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
					Start:     10 * telem.SecondTS,
					Authority: control.Absolute - 1,
					Subject:   control.Subject{Key: "foo"},
				}))
				Expect(t.Occurred()).To(BeTrue())
				Expect(MustSucceed(w1.Write(telem.NewSecondsTSV(0, 1, 2, 3, 4, 5)))).To(Equal(telem.Alignment(0)))
				w2, t := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
					Start:     10 * telem.SecondTS,
					Authority: control.Absolute,
					Subject:   control.Subject{Key: "bar"},
				}))
				Expect(t.Occurred()).To(BeTrue())
				Expect(MustSucceed(w2.Write(telem.NewSecondsTSV(6, 7, 8, 9, 10, 11)))).To(Equal(telem.Alignment(6)))
				a, err := w1.Write(telem.NewSecondsTSV(12, 13, 14, 15, 16, 17))
				Expect(err).To(MatchError(control.Unauthorized))
				Expect(a).To(Equal(telem.Alignment(0)))
				_, err = w1.Commit(ctx)
				Expect(err).To(MatchError(control.Unauthorized))
				t = MustSucceed(w2.Close())
				Expect(t.Occurred()).To(BeTrue())
				Expect(MustSucceed(w1.Write(telem.NewSecondsTSV(12, 13, 14, 15, 16, 17)))).To(Equal(telem.Alignment(12)))
				Expect(MustSucceed(w1.Commit(ctx))).To(Equal(17*telem.SecondTS + 1))
				t = MustSucceed(w1.Close())
				Expect(t.Occurred()).To(BeTrue())
			})
		})
	})

	Describe("Close", func() {
		var db = MustSucceed(unary.Open(unary.Config{
			FS: fs.NewMem(),
			Channel: core.Channel{
				Key:      100,
				DataType: telem.TimeStampT,
			},
		}))
		It("Should not allow operations on a closed writer", func() {
			var (
				w, t = MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
					Start:   10 * telem.SecondTS,
					Subject: control.Subject{Key: "foo"}},
				))
				e = core.EntityClosed("unary.writer")
			)
			Expect(t.Occurred()).To(BeTrue())
			_, err := w.Close()
			Expect(err).ToNot(HaveOccurred())
			_, err = w.Commit(ctx)
			Expect(err).To(MatchError(e))
			_, err = w.Write(telem.Series{Data: []byte{1, 2, 3}})
			Expect(err).To(MatchError(e))
			_, err = w.Close()
			Expect(err).To(BeNil())
			Expect(db.Close()).To(Succeed())
		})
	})
})
