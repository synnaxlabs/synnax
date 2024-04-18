package domain_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/testutil"
)

// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

var _ = Describe("File Controller", Ordered, func() {
	var db *domain.DB
	AfterEach(func() { Expect(db.Close()).To(Succeed()) })
	Describe("Writers", func() {
		It("Should allow one writing to a file at all times", func() {
			By("Initializing a file controller")
			db = testutil.MustSucceed(domain.Open(domain.Config{FS: fs.NewMem(), FileSize: 1 * telem.Megabyte}))
			By("Acquiring one writer on the file 1.domain")
			w1 := testutil.MustSucceed(db.NewWriter(ctx, domain.WriterConfig{
				Start: 10 * telem.SecondTS,
				End:   20 * telem.SecondTS,
			}))
			Expect(db.FS.Exists("1.domain"))
			By("Acquiring a second writer, this would create a new file 2.domain")
			w2, err := db.NewWriter(ctx, domain.WriterConfig{
				Start: 30 * telem.SecondTS,
				End:   40 * telem.SecondTS,
			})
			Expect(err).To(BeNil())
			Expect(db.FS.Exists("2.domain"))

			By("Closing the first writer")
			Expect(w1.Close()).To(Succeed())

			By("Acquiring a third writer, 1.domain should be acquired")
			w3, err := db.NewWriter(ctx, domain.WriterConfig{
				Start: 50 * telem.SecondTS,
				End:   60 * telem.SecondTS,
			})
			Expect(err).To(BeNil())
			n, err := w3.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})
			Expect(err).To(BeNil())
			Expect(n).To(Equal(8))
			s, err := db.FS.Stat("1.domain")
			Expect(err).To(BeNil())
			Expect(s.Size()).To(Equal(int64(8)))

			Expect(w2.Close()).To(Succeed())
			Expect(w3.Close()).To(Succeed())
		})

		It("Should obey the file size limit", func() {
			By("Initializing a file controller")
			db = testutil.MustSucceed(domain.Open(domain.Config{FS: fs.NewMem(), FileSize: 10 * telem.ByteSize}))
			By("Acquiring one writer on the file 1.domain")
			w1 := testutil.MustSucceed(db.NewWriter(ctx, domain.WriterConfig{
				Start: 10 * telem.SecondTS,
				End:   20 * telem.SecondTS,
			}))
			Expect(db.FS.Exists("1.domain"))
			n, err := w1.Write([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
			Expect(n).To(Equal(10))
			Expect(err).To(BeNil())
			Expect(w1.Close()).To(Succeed())
			By("Acquiring a second writer, this would create a new file 2.domain since 1.domain is full")
			w2, err := db.NewWriter(ctx, domain.WriterConfig{
				Start: 30 * telem.SecondTS,
				End:   40 * telem.SecondTS,
			})
			Expect(err).To(BeNil())
			Expect(db.FS.Exists("2.domain"))

			Expect(w2.Close()).To(Succeed())
		})

		It("Should obey the file descriptor limit", func() {
			By("Initializing a file controller")
			db = testutil.MustSucceed(domain.Open(domain.Config{FS: fs.NewMem(), MaxDescriptors: 2}))
			By("Acquiring one writer on the file 1.domain")
			w1 := testutil.MustSucceed(db.NewWriter(ctx, domain.WriterConfig{
				Start: 10 * telem.SecondTS,
				End:   20 * telem.SecondTS,
			}))
			Expect(db.FS.Exists("1.domain"))

			By("Acquiring one writer on the file 2.domain")
			w2 := testutil.MustSucceed(db.NewWriter(ctx, domain.WriterConfig{
				Start: 20 * telem.SecondTS,
				End:   30 * telem.SecondTS,
			}))
			Expect(db.FS.Exists("2.domain"))

			By("Trying to acquire a third writer")
			released := make(chan struct{})
			go func() {
				w3, err := db.NewWriter(ctx, domain.WriterConfig{
					Start: 30 * telem.SecondTS,
					End:   40 * telem.SecondTS,
				})
				Expect(err).To(BeNil())
				released <- struct{}{}
				Expect(w3.Close()).To(Succeed())
			}()
			By("Expecting it to block")
			Expect(len(released)).To(Equal(0))
			Expect(w1.Close()).To(Succeed())
			By("Expecting it to acquire")
			<-released
			Expect(w2.Close()).To(Succeed())
		})
	})
})
