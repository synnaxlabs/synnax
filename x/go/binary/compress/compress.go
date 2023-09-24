package compress

import (
	"math"
)

// a type called Compressor which is an interace
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

// Function Tested
func preCompile(src []byte) (size int) {
	longestCount, curCount, prev := 0, 0, byte(0)
	for _, x := range src {
		if x == prev {
			curCount++
		} else {
			curCount = 1
			prev = x
		}
		longestCount = int(math.Max(float64(longestCount), float64(curCount)))
	}

	returnCount := 0

	switch {
	case longestCount < 2:
		returnCount = 1
		break
	case longestCount < 4:
		returnCount = 2
		break
	case longestCount < 16:
		returnCount = 4
		break
	case longestCount < 256:
		returnCount = 8
		break
	case longestCount < 65536:
		returnCount = 16
		break
	case longestCount < 16777216:
		returnCount = 24
		break
	case longestCount < 4294967296:
		returnCount = 32
		break
	default:
		returnCount = -1
	}

	return returnCount
}

// Should start on 0
func Compress(src []byte) (dst []byte, err error) {
	count, appendVal, curShift := 0, 0, 0
	prev := byte(0)
	var returnArray []byte

	// first 4 bit stores size (1,2,4,8,16,24,32)
	// second 4 bit stores how many 0

	// This shouldn't need to be error checked.
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
			appendVal <<= (8 - curShift)
		}
		returnArray = append(returnArray, byte(appendVal))
	}

	return returnArray, nil
}

func Decompress(src []byte) (dst []byte, err error) {
	maxShift := int(src[0])
	cur := byte(0)
	// only used on > 8 but need to be global
	sum := 0
	count := 0

	var returnArray []byte

	for i := 1; i < len(src); i++ {
		if maxShift > 8 {
			sum += int(src[i])
			count += 8
			if count == maxShift {
				for j := 0; j < sum; j++ {
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
				for j := 0; j < int(src[i])&mask; j++ {
					returnArray = append(returnArray, cur)
				}
				mask >>= 2
				cur ^= 1
			}
		} else if maxShift == 4 {
			for j := 0; j < int(src[i])>>4; j++ {
				returnArray = append(returnArray, byte(0))
			}
			for j := 0; j < int(src[i])&0xF; j++ {
				returnArray = append(returnArray, byte(1))
			}
		} else if maxShift == 8 {
			for j := 0; j < int(src[i]); j++ {
				returnArray = append(returnArray, cur)
			}
			cur ^= 1
		}
	}

	return returnArray, nil
}
