// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package os_test

import (
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xos "github.com/synnaxlabs/x/os"
	. "github.com/synnaxlabs/x/testutil"
)

var _ io.ReadWriteCloser = xos.StdIO

var _ = Describe("StdIO", func() {
	It("Should return nil on Close", func() {
		Expect(xos.StdIO.Close()).To(Succeed())
	})

	It("Should delegate writes to os.Stdout", func() {
		r, w := MustSucceed2(os.Pipe())
		defer func() { Expect(r.Close()).To(Succeed()) }()

		origStdout := os.Stdout
		os.Stdout = w
		defer func() { os.Stdout = origStdout }()

		data := []byte("hello stdout")
		n := MustSucceed(xos.StdIO.Write(data))
		Expect(n).To(Equal(len(data)))

		Expect(w.Close()).To(Succeed())
		buf := make([]byte, 64)
		n = MustSucceed(r.Read(buf))
		Expect(string(buf[:n])).To(Equal("hello stdout"))
	})

	It("Should delegate reads to os.Stdin", func() {
		r, w := MustSucceed2(os.Pipe())
		defer func() { Expect(w.Close()).To(Succeed()) }()

		origStdin := os.Stdin
		os.Stdin = r
		defer func() { os.Stdin = origStdin }()

		data := []byte("hello stdin")
		MustSucceed(w.Write(data))

		buf := make([]byte, 64)
		n := MustSucceed(xos.StdIO.Read(buf))
		Expect(n).To(Equal(len(data)))
		Expect(string(buf[:n])).To(Equal("hello stdin"))
	})
})
