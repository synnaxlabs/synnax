// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package virtual_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/virtual"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/control"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Write", func() {
	var db *virtual.DB
	BeforeEach(func() {
		db = MustSucceed(virtual.Open(virtual.Config{
			MetaCodec: codec,
			Channel: core.Channel{
				Key:      2,
				DataType: telem.TimeStampT,
				Virtual:  true,
			},
			FS: xfs.NewMem(),
		}))
	})
	AfterEach(func() {
		Expect(db.Close()).To(Succeed())
	})
	Describe("Control", func() {
		Describe("ErrOnUnauthorized", func() {
			It("Should return an error if the writer does not acquire control", func() {
				w1, t := MustSucceed2(db.OpenWriter(ctx, virtual.WriterConfig{
					Start:             10 * telem.SecondTS,
					Authority:         control.Absolute,
					Subject:           control.Subject{Key: "foo"},
					ErrOnUnauthorized: config.True(),
				}))
				Expect(t.Occurred()).To(BeTrue())
				w2, t, err := db.OpenWriter(ctx, virtual.WriterConfig{
					Start:             10 * telem.SecondTS,
					Authority:         control.Absolute - 1,
					Subject:           control.Subject{Key: "bar"},
					ErrOnUnauthorized: config.True(),
				})
				Expect(err).To(HaveOccurredAs(control.Unauthorized))
				Expect(t.Occurred()).To(BeFalse())
				Expect(w2).To(BeNil())
				t = MustSucceed(w1.Close())
				Expect(t.Occurred()).To(BeTrue())
				Expect(t.IsRelease()).To(BeTrue())
			})
		})

		Describe("Write", func() {
			It("Should return an unauthorized error when the write is not authorized", func() {
				w1, t := MustSucceed2(db.OpenWriter(ctx, virtual.WriterConfig{
					Start:             10 * telem.SecondTS,
					Authority:         control.Absolute,
					Subject:           control.Subject{Key: "foo"},
					ErrOnUnauthorized: config.True(),
				}))
				Expect(t.Occurred()).To(BeTrue())
				w2, t := MustSucceed2(db.OpenWriter(ctx, virtual.WriterConfig{
					Start:     10 * telem.SecondTS,
					Authority: control.Absolute - 1,
					Subject:   control.Subject{Key: "bar"},
				}))
				Expect(t.Occurred()).To(BeFalse())
				_, err := w2.Write(telem.NewSecondsTSV(10, 11, 12))
				Expect(err).To(HaveOccurredAs(control.Unauthorized))
				MustSucceed(w1.Write(telem.NewSecondsTSV(10, 11, 12)))
				t = MustSucceed(w1.Close())
				Expect(t.Occurred()).To(BeTrue())
				t = MustSucceed(w2.Close())
				Expect(t.Occurred()).To(BeTrue())
			})

		})

		Describe("SetAuthority", func() {
			It("Should correctly set the authority of the writer", func() {

				w1, t := MustSucceed2(db.OpenWriter(ctx, virtual.WriterConfig{
					Start:             10 * telem.SecondTS,
					Authority:         control.Absolute - 2,
					Subject:           control.Subject{Key: "foo"},
					ErrOnUnauthorized: config.True(),
				}))
				Expect(t.Occurred()).To(BeTrue())
				w2, t := MustSucceed2(db.OpenWriter(ctx, virtual.WriterConfig{
					Start:     10 * telem.SecondTS,
					Authority: control.Absolute - 3,
					Subject:   control.Subject{Key: "bar"},
				}))
				Expect(t.Occurred()).To(BeFalse())
				_, err := w2.Write(telem.NewSecondsTSV(10, 11, 12))
				Expect(err).To(HaveOccurredAs(control.Unauthorized))
				t = w2.SetAuthority(control.Absolute - 1)
				Expect(t.Occurred()).To(BeTrue())
				MustSucceed(w2.Write(telem.NewSecondsTSV(10, 11, 12)))
				t = MustSucceed(w1.Close())
				Expect(t.Occurred()).To(BeFalse())
				t = MustSucceed(w2.Close())
				Expect(t.Occurred()).To(BeTrue())
			})
		})
	})
})
