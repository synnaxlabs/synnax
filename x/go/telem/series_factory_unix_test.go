// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build unix

package telem_test

import (
	"math"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Variable samples over 4 GiB", func() {
	var sample []byte
	BeforeEach(func() {
		// A read-only anonymous mapping takes no memory until a page is read.
		sample = MustSucceed(syscall.Mmap(
			-1,
			0,
			math.MaxUint32+1,
			syscall.PROT_READ,
			syscall.MAP_ANON|syscall.MAP_PRIVATE,
		))
		DeferCleanup(func() { Expect(syscall.Munmap(sample)).To(Succeed()) })
	})
	const msg = "telem: a variable sample of 4294967296 bytes is larger than the 4 GiB " +
		"limit"

	It("Should panic in MarshalVariableSample", func() {
		Expect(func() { telem.MarshalVariableSample(sample) }).To(PanicWith(msg))
	})

	It("Should panic in NewSeries", func() {
		Expect(func() { telem.NewSeriesV(sample) }).To(PanicWith(msg))
	})
})
