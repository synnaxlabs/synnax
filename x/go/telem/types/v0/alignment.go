// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"encoding/json"
	"fmt"
	"math"

	xjson "github.com/synnaxlabs/x/encoding/json"
)

var (
	_ json.Unmarshaler = (*Alignment)(nil)
	_ json.Marshaler   = (*Alignment)(nil)
)

// newAlignment packs a domain and sample index into an Alignment for the methods on
// this type. The exported constructor lives in the top-level telem package.
func newAlignment(domainIdx, sampleIdx uint32) Alignment {
	return Alignment(domainIdx)<<32 | Alignment(sampleIdx)
}

// MaxAlignment is the maximum possible value for an alignment.
const MaxAlignment = Alignment(math.MaxUint64)

// DomainIndex returns the domain index of the Alignment. This is the index
// in the array of arrays.
func (a Alignment) DomainIndex() uint32 { return uint32(a >> 32) }

// SampleIndex returns the sample index of the Alignment. This is the index within
// a particular array.
func (a Alignment) SampleIndex() uint32 { return uint32(a) }

// String implements fmt.Stringer to return a nicely formatted string representing the
// alignment.
func (a Alignment) String() string {
	return fmt.Sprintf("%v-%v", a.DomainIndex(), a.SampleIndex())
}

// UnmarshalJSON implements json.Unmarshaler.
func (a *Alignment) UnmarshalJSON(b []byte) error {
	n, err := xjson.UnmarshalStringUint64(b)
	if err != nil {
		return err
	}
	*a = Alignment(n)
	return nil
}

// MarshalJSON implements json.Marshaler.
func (a Alignment) MarshalJSON() ([]byte, error) {
	return xjson.MarshalStringUint64(uint64(a)), nil
}

// AddSamples increments the sample index of the alignment.
func (a Alignment) AddSamples(samples uint32) Alignment {
	return newAlignment(a.DomainIndex(), a.SampleIndex()+samples)
}
