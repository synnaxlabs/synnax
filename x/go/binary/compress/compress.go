// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compress

import (
	"math"
)

// Compressor a type called Compressor which is an interface
type Compressor interface {
	// Compress compresses the given source bytes and outputs them
	// to the destination bytes, returning any error encountered.
	Compress(src []byte) (dst []byte, err error)
}

type Decompressor interface {
	// Decompress decompresses the given source bytes and outputs
	// them to the destination bytes, returning any error encountered.
	Decompress(src []byte) (dst []byte, err error)
}

type CompressorDecompressor interface {
	Compressor
	Decompressor
}

/*
preCompile is run everytime Compress is run. It will do a pass through
the array, looking for the longest sequential count of either 1 or 0.
This will be used to decide on the size for the run-length encoding
*/
func preCompile(src []byte) (size int) {

	var (
		longestCount, curCount = 1, 1
	)

	for i := 1; i < len(src); i++ {
		if src[i] == src[i-1] {
			curCount++
		} else {
			if curCount > longestCount {
				longestCount = curCount
			}
			curCount = 1
		}
	}

	// Need to test final time because loop only checks in else
	if curCount > longestCount {
		longestCount = curCount
	}

	// returns how many bits are needed to store the longestCount
	return int(math.Pow(2, math.Floor(math.Log2(float64(longestCount)))))
}

type Bool struct{}

var _ CompressorDecompressor = Bool{}

func (b Bool) Compress(src []byte) (dst []byte, err error) {
	var (
		count, appendVal, curShift = 0, 0, 0
		prev                       = byte(0)
		returnArray                []byte
	)

	// Possible values include {1, 2, 4, 8, 16, 24, 32}
	maxShift := preCompile(src)
	maxSize := int(math.Pow(2, float64(maxShift))) - 1

	returnArray = append(returnArray, byte(maxShift))

	for _, x := range src {
		if x != prev || count == maxSize {
			if maxShift > 8 {
				for i := maxSize / 8; i > 0; i-- {
					returnArray = append(returnArray, byte(count&(0xFF<<i*8)))
				}
			} else {
				appendVal |= count
				curShift += maxShift
				if curShift == 8 {
					returnArray = append(returnArray, byte(appendVal))
					appendVal = 0
					curShift = 0
				}
				appendVal <<= maxShift
			}
			count = 1
		} else {
			count++
		}
		prev = x
	}

	if maxShift > 8 {
		for i := maxSize / 8; i > 0; i-- {
			returnArray = append(returnArray, byte(count&(0xFF<<i*8)))
		}
	} else {
		appendVal |= count
		curShift += maxShift
		if curShift < 8 {
			appendVal <<= 8 - curShift
		}
		returnArray = append(returnArray, byte(appendVal))
	}

	return returnArray, nil
}

func (b Bool) Decompress(src []byte) (dst []byte, err error) {

	var (
		maxShift, sum, count = int(src[0]), 0, 0
		cur                  = byte(0)
		returnArray          []byte
	)

	for i := 1; i < len(src); i++ {
		if maxShift > 8 {
			sum += int(src[i])
			count += 8
			if count == maxShift {
				for range sum {
					returnArray = append(returnArray, cur)
				}
				count, sum = 0, 0
				cur ^= 1
			}
			sum <<= 8
		} else if maxShift == 1 {
			for j := 7; j >= 0; j-- {
				value := int(src[i] >> j)
				if value&1 == 1 {
					returnArray = append(returnArray, cur)
				}
				cur ^= 1
			}
		} else if maxShift == 2 {
			mask := 192
			for mask > 0 {
				for range int(src[i]) & mask {
					returnArray = append(returnArray, cur)
				}
				mask >>= 2
				cur ^= 1
			}
		} else if maxShift == 4 {
			for range int(src[i]) >> 4 {
				returnArray = append(returnArray, byte(0))
			}
			for range int(src[i]) & 0xF {
				returnArray = append(returnArray, byte(1))
			}
		} else if maxShift == 8 {
			for range int(src[i]) {
				returnArray = append(returnArray, cur)
			}
			cur ^= 1
		}
	}

	return returnArray, nil
}
