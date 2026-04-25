// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fs_test

import (
	"os"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Recorder", func() {
	var rec *xfs.Recorder
	BeforeEach(func() { rec = xfs.NewRecorder(xfs.NewMem()) })

	Describe("Open", func() {
		It("Should record a successful open", func() {
			f := MustSucceed(rec.Open("a.bin", os.O_CREATE|os.O_RDWR))
			DeferClose(f)
			Expect(rec.EventsFor("a.bin")).To(ConsistOf(
				xfs.Event{Op: xfs.OpOpen, Name: "a.bin"},
			))
		})

		It("Should not record a failed open", func() {
			Expect(rec.Open("missing.bin", os.O_RDONLY)).Error().To(HaveOccurred())
			Expect(rec.Events()).To(BeEmpty())
		})
	})

	Describe("File operations", func() {
		var f xfs.File
		BeforeEach(func() {
			f = MustSucceed(rec.Open("a.bin", os.O_CREATE|os.O_RDWR))
			DeferClose(f)
			rec.Reset()
		})

		It("Should record Write with the requested length", func() {
			MustSucceed(f.Write([]byte("hello")))
			Expect(rec.EventsFor("a.bin")).To(ConsistOf(
				xfs.Event{Op: xfs.OpWrite, Name: "a.bin", Length: 5},
			))
		})

		It("Should record WriteAt with offset and length", func() {
			MustSucceed(f.WriteAt([]byte("hi"), 7))
			Expect(rec.EventsFor("a.bin")).To(ConsistOf(
				xfs.Event{Op: xfs.OpWriteAt, Name: "a.bin", Offset: 7, Length: 2},
			))
		})

		It("Should record Read with the requested length", func() {
			MustSucceed(f.Write([]byte("hello")))
			rec.Reset()
			buf := make([]byte, 5)
			MustSucceed(f.Read(buf))
			Expect(rec.EventsFor("a.bin")).To(ConsistOf(
				xfs.Event{Op: xfs.OpRead, Name: "a.bin", Length: 5},
			))
		})

		It("Should record ReadAt with offset and length", func() {
			MustSucceed(f.WriteAt([]byte("hello"), 0))
			rec.Reset()
			buf := make([]byte, 4)
			MustSucceed(f.ReadAt(buf, 1))
			Expect(rec.EventsFor("a.bin")).To(ConsistOf(
				xfs.Event{Op: xfs.OpReadAt, Name: "a.bin", Offset: 1, Length: 4},
			))
		})

		It("Should record events in the order they occurred", func() {
			MustSucceed(f.Write([]byte("ab")))
			MustSucceed(f.WriteAt([]byte("cd"), 4))
			MustSucceed(f.ReadAt(make([]byte, 1), 0))
			Expect(rec.EventsFor("a.bin")).To(Equal([]xfs.Event{
				{Op: xfs.OpWrite, Name: "a.bin", Length: 2},
				{Op: xfs.OpWriteAt, Name: "a.bin", Offset: 4, Length: 2},
				{Op: xfs.OpReadAt, Name: "a.bin", Offset: 0, Length: 1},
			}))
		})
	})

	Describe("Sub", func() {
		It("Should record events from sub-derived FS instances into the parent log", func() {
			sub := MustSucceed(rec.Sub("nested"))
			f := MustSucceed(sub.Open("a.bin", os.O_CREATE|os.O_RDWR))
			DeferClose(f)
			MustSucceed(f.Write([]byte("xyz")))
			Expect(rec.Events()).To(Equal([]xfs.Event{
				{Op: xfs.OpOpen, Name: "a.bin"},
				{Op: xfs.OpWrite, Name: "a.bin", Length: 3},
			}))
		})

		It("Should share the event log across nested Subs", func() {
			subA := MustSucceed(rec.Sub("a"))
			subB := MustSucceed(subA.Sub("b"))
			f := MustSucceed(subB.Open("file.bin", os.O_CREATE|os.O_RDWR))
			DeferClose(f)
			Expect(rec.EventsFor("file.bin")).To(HaveLen(1))
		})

		It("Should reflect Reset across the parent and every sub", func() {
			sub := MustSucceed(rec.Sub("nested"))
			f := MustSucceed(sub.Open("a.bin", os.O_CREATE|os.O_RDWR))
			DeferClose(f)
			rec.Reset()
			Expect(sub.(*xfs.Recorder).Events()).To(BeEmpty())
		})
	})

	Describe("EventsFor", func() {
		It("Should return only events targeting the named file", func() {
			a := MustSucceed(rec.Open("a.bin", os.O_CREATE|os.O_RDWR))
			DeferClose(a)
			b := MustSucceed(rec.Open("b.bin", os.O_CREATE|os.O_RDWR))
			DeferClose(b)
			MustSucceed(a.Write([]byte("a")))
			MustSucceed(b.Write([]byte("bb")))
			Expect(rec.EventsFor("a.bin")).To(Equal([]xfs.Event{
				{Op: xfs.OpOpen, Name: "a.bin"},
				{Op: xfs.OpWrite, Name: "a.bin", Length: 1},
			}))
			Expect(rec.EventsFor("b.bin")).To(Equal([]xfs.Event{
				{Op: xfs.OpOpen, Name: "b.bin"},
				{Op: xfs.OpWrite, Name: "b.bin", Length: 2},
			}))
		})

		It("Should return an empty slice for an unknown file", func() {
			Expect(rec.EventsFor("missing.bin")).To(BeEmpty())
		})
	})

	Describe("Reset", func() {
		It("Should clear every recorded event", func() {
			f := MustSucceed(rec.Open("a.bin", os.O_CREATE|os.O_RDWR))
			DeferClose(f)
			MustSucceed(f.Write([]byte("hello")))
			Expect(rec.Events()).ToNot(BeEmpty())
			rec.Reset()
			Expect(rec.Events()).To(BeEmpty())
		})
	})

	Describe("Pass-through", func() {
		It("Should pass non-recorded FS methods through to the inner FS without recording", func() {
			f := MustSucceed(rec.Open("a.bin", os.O_CREATE|os.O_RDWR))
			DeferClose(f)
			MustSucceed(f.Write([]byte("hello")))
			rec.Reset()

			Expect(rec.Exists("a.bin")).To(BeTrue())
			Expect(rec.Exists("missing.bin")).To(BeFalse())
			Expect(rec.Events()).To(BeEmpty())
		})

		It("Should pass non-recorded File methods through to the inner File without recording", func() {
			f := MustSucceed(rec.Open("a.bin", os.O_CREATE|os.O_RDWR))
			DeferClose(f)
			MustSucceed(f.Write([]byte("hello")))
			rec.Reset()

			Expect(f.Truncate(2)).To(Succeed())
			Expect(f.Sync()).To(Succeed())
			Expect(rec.Events()).To(BeEmpty())
		})
	})

	Describe("Intent-based recording", func() {
		It("Should record the requested length on a short read", func() {
			f := MustSucceed(rec.Open("a.bin", os.O_CREATE|os.O_RDWR))
			DeferClose(f)
			MustSucceed(f.Write([]byte("hi")))
			rec.Reset()

			// Asking for more bytes than the file contains is a short read; the
			// recorder should still record the requested length, not the
			// returned length.
			buf := make([]byte, 16)
			_, _ = f.ReadAt(buf, 0)
			Expect(rec.EventsFor("a.bin")).To(ConsistOf(
				xfs.Event{Op: xfs.OpReadAt, Name: "a.bin", Offset: 0, Length: 16},
			))
		})

		It("Should record the requested length when ReadAt fails past EOF", func() {
			f := MustSucceed(rec.Open("a.bin", os.O_CREATE|os.O_RDWR))
			DeferClose(f)
			MustSucceed(f.Write([]byte("hi")))
			rec.Reset()

			// Reading past the end of the file fails, but the recorder should
			// still capture the attempt with the caller-requested length.
			buf := make([]byte, 4)
			_, _ = f.ReadAt(buf, 100)
			Expect(rec.EventsFor("a.bin")).To(ConsistOf(
				xfs.Event{Op: xfs.OpReadAt, Name: "a.bin", Offset: 100, Length: 4},
			))
		})
	})

	Describe("Concurrency", func() {
		It("Should record every event safely under concurrent writers", func() {
			const writers = 16
			const writesPerWriter = 64
			f := MustSucceed(rec.Open("a.bin", os.O_CREATE|os.O_RDWR))
			DeferClose(f)
			rec.Reset()

			var wg sync.WaitGroup
			wg.Add(writers)
			for range writers {
				go func() {
					defer wg.Done()
					payload := []byte("x")
					for range writesPerWriter {
						_, _ = f.WriteAt(payload, 0)
					}
				}()
			}
			wg.Wait()

			Expect(rec.EventsFor("a.bin")).To(HaveLen(writers * writesPerWriter))
		})
	})
})
