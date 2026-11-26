// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build unix

package codec_test

import (
	"io"
	"math"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Series over 4 GiB", func() {
	It(
		"Should fail to encode a variable series whose size the length field "+
			"cannot hold",
		func(
			ctx SpecContext,
		) {
			// A read-only anonymous mapping takes no memory until a page is read.
			data := MustSucceed(syscall.Mmap(
				-1,
				0,
				math.MaxUint32+1,
				syscall.PROT_READ,
				syscall.MAP_ANON|syscall.MAP_PRIVATE,
			))
			DeferCleanup(func() { Expect(syscall.Munmap(data)).To(Succeed()) })
			c := codec.NewStatic(channel.Keys{6}, []telem.DataType{telem.StringT})
			Expect(c.Encode(ctx, multi(keyed{6, telem.Series{
				DataType: telem.StringT,
				Data:     data,
			}}))).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"series of channel 6 holds 4294967296 bytes, more than a "+
						"frame length field holds",
				)),
			))
		},
	)

	It(
		"Should encode two adjacent series of one key that each fit the length field",
		func(ctx SpecContext) {
			const half = 1 << 31
			// A read-only anonymous mapping takes no memory until a page is read.
			data := MustSucceed(syscall.Mmap(
				-1,
				0,
				2*half,
				syscall.PROT_READ,
				syscall.MAP_ANON|syscall.MAP_PRIVATE,
			))
			DeferCleanup(func() { Expect(syscall.Munmap(data)).To(Succeed()) })
			c := codec.NewStatic(channel.Keys{6}, []telem.DataType{telem.Uint8T})
			// Each series holds 2^31 samples. Their alignments are contiguous, so the
			// encoder merges them into one series of 2^32 samples.
			Expect(c.EncodeStream(ctx, io.Discard, multi(
				keyed{6, telem.Series{DataType: telem.Uint8T, Data: data[:half]}},
				keyed{6, telem.Series{
					DataType:  telem.Uint8T,
					Data:      data[half:],
					Alignment: half,
				}},
			))).To(Succeed())
		},
	)

	It(
		"Should encode a run of series of one key whose pairs fit the length field "+
			"and whose sum does not",
		func(ctx SpecContext) {
			const size = 1<<32 + 1
			data := MustSucceed(syscall.Mmap(
				-1,
				0,
				size,
				syscall.PROT_READ,
				syscall.MAP_ANON|syscall.MAP_PRIVATE,
			))
			DeferCleanup(func() { Expect(syscall.Munmap(data)).To(Succeed()) })
			c := codec.NewStatic(channel.Keys{6}, []telem.DataType{telem.Uint8T})
			series := func(lo, hi int) telem.Series {
				return telem.Series{
					DataType:  telem.Uint8T,
					Data:      data[lo:hi],
					Alignment: telem.Alignment(lo),
				}
			}
			Expect(c.EncodeStream(ctx, io.Discard, multi(
				keyed{6, series(0, 2)},
				keyed{6, series(2, 4)},
				keyed{6, series(4, size)},
			))).To(Succeed())
		},
	)
})
