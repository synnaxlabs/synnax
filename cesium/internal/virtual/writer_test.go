// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/cesium/internal/virtual"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Write", func() {
	var db *virtual.DB
	BeforeEach(func() {
		db = MustSucceed(virtual.Open(ctx, virtual.Config{
			MetaCodec: codec,
			Channel: channel.Channel{
				Name:     "Ray",
				Key:      2,
				DataType: telem.TimeStampT,
				Virtual:  true,
			},
			FS: fs.NewMem(),
		}))
	})
	AfterEach(func() {
		Expect(db.Close()).To(Succeed())
	})
	Describe("Control", func() {
		Describe("ErrOnUnauthorizedOpen", func() {
			It("Should return an error if the writer does not acquire control", func() {
				w1, t := MustSucceed2(db.OpenWriter(ctx, virtual.WriterConfig{
					Start:                 10 * telem.SecondTS,
					Authority:             control.AuthorityAbsolute,
					Subject:               control.Subject{Key: "foo"},
					ErrOnUnauthorizedOpen: config.True(),
				}))
				Expect(t.Occurred()).To(BeTrue())
				w2, t, err := db.OpenWriter(ctx, virtual.WriterConfig{
					Start:                 10 * telem.SecondTS,
					Authority:             control.AuthorityAbsolute - 1,
					Subject:               control.Subject{Key: "bar"},
					ErrOnUnauthorizedOpen: config.True(),
				})
				Expect(err).To(HaveOccurredAs(control.ErrUnauthorized))
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
					Start:                 10 * telem.SecondTS,
					Authority:             control.AuthorityAbsolute,
					Subject:               control.Subject{Key: "foo"},
					ErrOnUnauthorizedOpen: config.True(),
				}))
				Expect(t.Occurred()).To(BeTrue())
				w2, t := MustSucceed2(db.OpenWriter(ctx, virtual.WriterConfig{
					Start:     10 * telem.SecondTS,
					Authority: control.AuthorityAbsolute - 1,
					Subject:   control.Subject{Key: "bar"},
				}))
				Expect(t.Occurred()).To(BeFalse())
				_, err := w2.Write(telem.NewSeriesSecondsTSV(10, 11, 12))
				Expect(err).To(HaveOccurredAs(control.ErrUnauthorized))
				MustSucceed(w1.Write(telem.NewSeriesSecondsTSV(10, 11, 12)))
				t = MustSucceed(w1.Close())
				Expect(t.Occurred()).To(BeTrue())
				t = MustSucceed(w2.Close())
				Expect(t.Occurred()).To(BeTrue())
			})

			It("Should return an error when writing a series with the wrong data type", func() {
				w, t := MustSucceed2(db.OpenWriter(ctx, virtual.WriterConfig{
					Start:     10 * telem.SecondTS,
					Authority: control.AuthorityAbsolute,
					Subject:   control.Subject{Key: "foo"},
				}))
				Expect(t.Occurred()).To(BeTrue())
				_, err := w.Write(telem.NewSeriesV[uint8](1, 2, 3))
				Expect(err).To(HaveOccurredAs(validate.Error))
				t = MustSucceed(w.Close())
				Expect(t.Occurred()).To(BeTrue())
			})

		})

		Describe("Close", func() {
			It("Should not return an error when the same writer is closed multiple times", func() {
				w, t := MustSucceed2(db.OpenWriter(ctx, virtual.WriterConfig{
					Start:     10 * telem.SecondTS,
					Authority: control.AuthorityAbsolute,
					Subject:   control.Subject{Key: "foo"},
				}))
				Expect(t.Occurred()).To(BeTrue())
				t = MustSucceed(w.Close())
				Expect(t.Occurred()).To(BeTrue())
				t = MustSucceed(w.Close())
				Expect(t.Occurred()).To(BeFalse())
			})

			It("Should return an error on Write when the DB is closed", func() {
				w, t := MustSucceed2(db.OpenWriter(ctx, virtual.WriterConfig{
					Start:     10 * telem.SecondTS,
					Authority: control.AuthorityAbsolute,
					Subject:   control.Subject{Key: "foo"},
				}))
				Expect(t.Occurred()).To(BeTrue())
				t = MustSucceed(w.Close())
				Expect(t.Occurred()).To(BeTrue())
				_, err := w.Write(telem.NewSeriesSecondsTSV(10, 11, 12))
				Expect(err).To(HaveOccurredAs(resource.NewErrClosed("virtual.writer")))
			})

		})

		Describe("SetAuthority", func() {
			It("Should correctly set the authority of the writer", func() {
				w1, t := MustSucceed2(db.OpenWriter(ctx, virtual.WriterConfig{
					Start:                 10 * telem.SecondTS,
					Authority:             control.AuthorityAbsolute - 2,
					Subject:               control.Subject{Key: "foo"},
					ErrOnUnauthorizedOpen: config.True(),
				}))
				Expect(t.Occurred()).To(BeTrue())

				w2, t := MustSucceed2(db.OpenWriter(ctx, virtual.WriterConfig{
					Start:     10 * telem.SecondTS,
					Authority: control.AuthorityAbsolute - 3,
					Subject:   control.Subject{Key: "bar"},
				}))

				Expect(t.Occurred()).To(BeFalse())

				_, err := w2.Write(telem.NewSeriesSecondsTSV(10, 11, 12))
				Expect(err).To(HaveOccurredAs(control.ErrUnauthorized))
				t = w2.SetAuthority(control.AuthorityAbsolute - 1)
				Expect(t.Occurred()).To(BeTrue())

				MustSucceed(w2.Write(telem.NewSeriesSecondsTSV(10, 11, 12)))
				t = MustSucceed(w1.Close())
				Expect(t.Occurred()).To(BeFalse())

				t = MustSucceed(w2.Close())
				Expect(t.Occurred()).To(BeTrue())
			})
		})
	})
})
