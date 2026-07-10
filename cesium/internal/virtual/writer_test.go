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
	"github.com/synnaxlabs/cesium/internal/virtual"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Write", func() {
	var db *virtual.DB
	BeforeEach(func() {
		db = MustOpen(virtual.Open(virtual.Config{
			Channel: channel.Channel{
				Name:     "Ray",
				Key:      2,
				DataType: telem.TimeStampT,
				Virtual:  true,
			},
		}))
	})
	Describe("OpenWriter", func() {
		It("Should return an error when opening a writer on a closed DB", func() {
			closedDB := MustSucceed(virtual.Open(virtual.Config{
				Channel: channel.Channel{
					Name:     "Egon",
					Key:      3,
					DataType: telem.Int64T,
					Virtual:  true,
				},
			}))
			Expect(closedDB.Close()).To(Succeed())
			Expect(closedDB.OpenWriter(virtual.WriterConfig{
				Subject: control.Subject{Key: "foo"},
			})).Error().To(MatchError(virtual.ErrDBClosed))
		})
	})

	Describe("Channel", func() {
		It("Should return the channel the writer writes to", func() {
			w, t := MustSucceed2(db.OpenWriter(virtual.WriterConfig{
				Start:     10 * telem.SecondTS,
				Authority: control.AuthorityAbsolute,
				Subject:   control.Subject{Key: "foo"},
			}))
			Expect(t.Occurred()).To(BeTrue())
			Expect(w.Channel()).To(Equal(db.Channel()))
			Expect(w.Channel().Name).To(Equal("Ray"))
			t = MustSucceed(w.Close())
			Expect(t.Occurred()).To(BeTrue())
		})
	})

	Describe("Control", func() {
		Describe("ErrOnUnauthorizedOpen", func() {
			It("Should return an error if the writer does not acquire control", func() {
				w1, t := MustSucceed2(db.OpenWriter(virtual.WriterConfig{
					Start:                 10 * telem.SecondTS,
					Authority:             control.AuthorityAbsolute,
					Subject:               control.Subject{Key: "foo"},
					ErrOnUnauthorizedOpen: new(true),
				}))
				Expect(t.Occurred()).To(BeTrue())
				w2, t, err := db.OpenWriter(virtual.WriterConfig{
					Start:                 10 * telem.SecondTS,
					Authority:             control.AuthorityAbsolute - 1,
					Subject:               control.Subject{Key: "bar"},
					ErrOnUnauthorizedOpen: new(true),
				})
				Expect(err).To(MatchError(control.ErrUnauthorized))
				Expect(t.Occurred()).To(BeFalse())
				Expect(w2).To(BeNil())
				t = MustSucceed(w1.Close())
				Expect(t.Occurred()).To(BeTrue())
				Expect(t.IsRelease()).To(BeTrue())
			})
		})

		Describe("Write", func() {
			It("Should return an unauthorized error when the write is not authorized", func() {
				w1, t := MustSucceed2(db.OpenWriter(virtual.WriterConfig{
					Start:                 10 * telem.SecondTS,
					Authority:             control.AuthorityAbsolute,
					Subject:               control.Subject{Key: "foo"},
					ErrOnUnauthorizedOpen: new(true),
				}))
				Expect(t.Occurred()).To(BeTrue())
				w2, t := MustSucceed2(db.OpenWriter(virtual.WriterConfig{
					Start:     10 * telem.SecondTS,
					Authority: control.AuthorityAbsolute - 1,
					Subject:   control.Subject{Key: "bar"},
				}))
				Expect(t.Occurred()).To(BeFalse())
				Expect(w2.Write(telem.NewSeriesSecondsTSV(10, 11, 12))).
					Error().To(MatchError(control.ErrUnauthorized))
				MustSucceed(w1.Write(telem.NewSeriesSecondsTSV(10, 11, 12)))
				t = MustSucceed(w1.Close())
				Expect(t.Occurred()).To(BeTrue())
				t = MustSucceed(w2.Close())
				Expect(t.Occurred()).To(BeTrue())
			})

			It("Should return an error when writing a series with the wrong data type", func() {
				w, t := MustSucceed2(db.OpenWriter(virtual.WriterConfig{
					Start:     10 * telem.SecondTS,
					Authority: control.AuthorityAbsolute,
					Subject:   control.Subject{Key: "foo"},
				}))
				Expect(t.Occurred()).To(BeTrue())
				Expect(w.Write(telem.NewSeriesV[uint8](1, 2, 3))).
					Error().To(MatchError(validate.ErrValidation))
				t = MustSucceed(w.Close())
				Expect(t.Occurred()).To(BeTrue())
			})

		})

		Describe("Close", func() {
			It("Should not return an error when the same writer is closed multiple times", func() {
				w, t := MustSucceed2(db.OpenWriter(virtual.WriterConfig{
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
				w, t := MustSucceed2(db.OpenWriter(virtual.WriterConfig{
					Start:     10 * telem.SecondTS,
					Authority: control.AuthorityAbsolute,
					Subject:   control.Subject{Key: "foo"},
				}))
				Expect(t.Occurred()).To(BeTrue())
				t = MustSucceed(w.Close())
				Expect(t.Occurred()).To(BeTrue())
				Expect(w.Write(telem.NewSeriesSecondsTSV(10, 11, 12))).
					Error().To(MatchError(virtual.ErrWriterClosed))
			})

		})

		Describe("SetAuthority", func() {
			It("Should correctly set the authority of the writer", func() {
				w1, t := MustSucceed2(db.OpenWriter(virtual.WriterConfig{
					Start:                 10 * telem.SecondTS,
					Authority:             control.AuthorityAbsolute - 2,
					Subject:               control.Subject{Key: "foo"},
					ErrOnUnauthorizedOpen: new(true),
				}))
				Expect(t.Occurred()).To(BeTrue())

				w2, t := MustSucceed2(db.OpenWriter(virtual.WriterConfig{
					Start:     10 * telem.SecondTS,
					Authority: control.AuthorityAbsolute - 3,
					Subject:   control.Subject{Key: "bar"},
				}))
				Expect(t.Occurred()).To(BeFalse())

				Expect(w2.Write(telem.NewSeriesSecondsTSV(10, 11, 12))).
					Error().To(MatchError(control.ErrUnauthorized))
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
