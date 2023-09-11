package compress

import (
    "math"
)

// a type called Compressor which is an interace
type Compressor interface {
    // Compress compresses the given source bytes and outputs them
    // to the destination bytes, returning any error encountered.
    Compress(src []byte) (dst []byte, err error)
    CompressUp(src []byte) (dst []byte, err error) 
}

type Decompressor interface {
    // Decompress decompresses the given source bytes and outputs
    // them to the destination bytes, returning any error encountered.
    Decompress(src []byte) (dst []byte, err error)
    DecompressUp(src []byte) (dst []byte, err error)
}

// Function Tested
func preCompile(src []byte) (size int) {
    longestCount, curCount, prev := 0, 0, byte(0)
    for _, x := range src {
        if (x == prev) {
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
func CompressUp(src []byte) (dst []byte, size int, err error) {
    count, appendVal, curShift := 0, 0, 0
    prev := byte(0)
    var returnArray []byte

    // This shouldn't need to be error checked. 
    // Possible values include {1, 2, 4, 8, 16, 24, 32}
    maxShift := preCompile(src);
    maxSize := int(math.Pow(2, float64(maxShift))) - 1

    for _, x := range src {
        if x != prev || count == maxSize {
            if (maxShift > 8) {
                for i := maxSize % 8; i > 0; i-- {
                    returnArray = append(returnArray, byte(count & (0xFF << i * 8)))
                }
            } else {
                appendVal |= count
                appendVal <<= maxShift
                curShift += maxShift
                if (curShift == 8) {
                    returnArray = append(returnArray, byte(appendVal))
                    appendVal = 0
                    curShift = 0
                }
            }
            count = 1
        } else {
            count++
        }
        prev = x
    }

    if (count > 1) {
        returnArray = append(returnArray, byte(appendVal))
    } else {
        returnArray = append(returnArray, byte(appendVal))
    }

    return returnArray, maxShift, nil
}

func DecompressUp(src []byte, size int) (dst []byte, err error) {

    var returnArray []byte

    for _, x := range src {
        zeroCount := (x & 0xF0) >> 4
        oneCount := (x & 0x0F)
        for i := 0; i < int(zeroCount); i++ {
            returnArray = append(returnArray, 0)
        }
        for i := 0; i < int(oneCount); i++ {
            returnArray = append(returnArray, 1)
        }
    }

    return returnArray, nil
}
