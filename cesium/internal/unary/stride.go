// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unary

import (
	"bufio"
	"io"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// strideBufferSize bounds the scratch buffer a strided read holds, so the buffer
// never scales with the size of the slice being read.
const strideBufferSize = 64 * telem.Kilobyte

// readStrided reads every factor-th sample of the slice [offset, offset+size) in r,
// packing the kept samples into a buffer sized to them alone. It returns the packed
// data and the number of source samples the slice held. A slice shorter than size
// yields the samples that were available.
func readStrided(
	r io.ReaderAt,
	dt telem.DataType,
	offset telem.Size,
	size telem.Size,
	factor int64,
) ([]byte, int64, error) {
	if dt.IsVariable() {
		return readStridedVariable(r, offset, size, factor)
	}
	return readStridedFixed(r, dt.Density(), offset, size, factor)
}

func readStridedFixed(
	r io.ReaderAt,
	density telem.Density,
	offset telem.Size,
	size telem.Size,
	factor int64,
) ([]byte, int64, error) {
	var (
		srcSamples = density.SampleCount(size)
		kept       = srcSamples/factor + min(srcSamples%factor, 1)
		stride     = int64(density.Size(factor))
		// batch is how many kept samples a single ReadAt covers. A stride wider than
		// the buffer drops it to one, so the read skips the discarded samples instead
		// of pulling them through the buffer.
		batch = max(int64(strideBufferSize)/stride, 1)
		buf   = make([]byte, stride*(batch-1)+int64(density))
		out   = make([]byte, 0, density.Size(kept))
	)
	for read := int64(0); read < kept; {
		n := min(batch, kept-read)
		b := buf[:stride*(n-1)+int64(density)]
		count, err := r.ReadAt(b, int64(offset+density.Size(read*factor)))
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, 0, err
		}
		avail := density.SampleCount(telem.Size(count))
		for j := int64(0); j < n && j*factor < avail; j++ {
			out = append(out, b[j*stride:j*stride+int64(density)]...)
		}
		if count < len(b) {
			return out, read*factor + avail, nil
		}
		read += n
	}
	return out, srcSamples, nil
}

func readStridedVariable(
	r io.ReaderAt,
	offset telem.Size,
	size telem.Size,
	factor int64,
) ([]byte, int64, error) {
	var (
		br = bufio.NewReaderSize(
			io.NewSectionReader(r, int64(offset), int64(size)),
			int(min(strideBufferSize, size)),
		)
		lenBuf = make([]byte, 4)
		out    []byte
		src    int64
	)
	for pos := int64(0); pos+4 <= int64(size); src++ {
		if _, err := io.ReadFull(br, lenBuf); err != nil {
			if errors.IsAny(err, io.EOF, io.ErrUnexpectedEOF) {
				return out, src, nil
			}
			return nil, 0, err
		}
		length := int64(telem.ByteOrder.Uint32(lenBuf))
		// A length prefix is stored data. Without this bound a corrupt one would
		// drive an allocation of up to 4GiB before the short read caught it.
		if pos+4+length > int64(size) {
			return out, src, nil
		}
		pos += 4 + length
		if src%factor != 0 {
			if _, err := br.Discard(int(length)); err != nil {
				if errors.Is(err, io.EOF) {
					return out, src, nil
				}
				return nil, 0, err
			}
			continue
		}
		out = append(out, lenBuf...)
		start := len(out)
		out = append(out, make([]byte, length)...)
		if _, err := io.ReadFull(br, out[start:]); err != nil {
			if errors.IsAny(err, io.EOF, io.ErrUnexpectedEOF) {
				return out[:start-4], src, nil
			}
			return nil, 0, err
		}
	}
	return out, src, nil
}
