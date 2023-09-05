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
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("TypedWriter Behavior", func() {
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
			w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
				Start: telem.TimeStamp(0),
			}))
			Expect(w.Write(telem.NewSecondsTSV(0, 1, 2, 3, 4, 5))).To(Succeed())
			Expect(w.Write(telem.NewSecondsTSV(6, 7, 8, 9, 10, 11))).To(Succeed())
			Expect(MustSucceed(w.Commit(ctx))).To(Equal(11*telem.SecondTS + 1))
			Expect(w.Close()).To(Succeed())
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
			w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{Start: 10 * telem.SecondTS}))
			Expect(w.Write(telem.NewSeries([]int64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}))).To(Succeed())
			Expect(MustSucceed(w.Commit(ctx))).To(Equal(20*telem.SecondTS + 1))
			Expect(w.Close()).To(Succeed())
		})
	})
})
