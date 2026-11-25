// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package codec_test

import (
	"encoding/hex"
	"encoding/json/v2"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	. "github.com/synnaxlabs/x/testutil"
)

type fuzzDecoded struct {
	Key       uint32   `json:"key"`
	DataType  string   `json:"data_type"`
	Data      string   `json:"data"`
	Clock     uint32   `json:"clock"`
	TimeRange [2]int64 `json:"time_range"`
	Alignment uint64   `json:"alignment"`
}

type fuzzFrame struct {
	Name    string         `json:"name"`
	Series  []vectorSeries `json:"series"`
	Encoded string         `json:"encoded"`
	Error   string         `json:"error"`
	Decoded []fuzzDecoded  `json:"decoded"`
}

type fuzzCase struct {
	Name      string      `json:"name"`
	Keys      []uint32    `json:"keys"`
	DataTypes []string    `json:"data_types"`
	Frames    []fuzzFrame `json:"frames"`
}

type truncResult struct {
	Case    string        `json:"case"`
	Frame   int           `json:"frame"`
	Cut     int           `json:"cut"`
	Error   string        `json:"error"`
	Decoded []fuzzDecoded `json:"decoded"`
}

var _ = Describe("Scratch trunc", func() {
	It("Should decode every truncation of the golden frames", func(ctx SpecContext) {
		out := os.Getenv("TRUNC_OUT")
		if out == "" {
			Skip("no TRUNC_OUT")
		}
		var file vectorFile
		Expect(json.Unmarshal(MustSucceed(os.ReadFile(vectorsPath)), &file)).To(Succeed())
		var results []truncResult
		for _, vc := range file.Cases {
			for fi, f := range vc.Frames {
				full := MustSucceed(hex.DecodeString(f.Encoded))
				for cut := 0; cut < len(full); cut++ {
					dec := codec.NewStatic(vc.Keys, vc.DataTypes)
					for _, prev := range vc.Frames[:fi] {
						MustSucceed(dec.Decode(MustSucceed(hex.DecodeString(prev.Encoded))))
					}
					r := truncResult{Case: vc.Name, Frame: fi, Cut: cut}
					fr, err := dec.Decode(full[:cut])
					if err != nil {
						r.Error = err.Error()
					} else {
						for k, s := range fr.Entries() {
							r.Decoded = append(r.Decoded, fuzzDecoded{
								Key:      uint32(k),
								DataType: string(s.DataType),
								Data:     hex.EncodeToString(s.Data),
								Clock:    s.Clock,
							})
						}
					}
					results = append(results, r)
				}
			}
		}
		b := MustSucceed(json.Marshal(map[string]any{"results": results}))
		Expect(os.WriteFile(out, b, 0o644)).To(Succeed())
	})
})

var _ = Describe("Scratch fuzz", func() {
	It("Should encode the scratch cases", func(ctx SpecContext) {
		in := os.Getenv("FUZZ_IN")
		if in == "" {
			Skip("no FUZZ_IN")
		}
		var file struct {
			Cases []vectorCase `json:"cases"`
		}
		Expect(json.Unmarshal(MustSucceed(os.ReadFile(in)), &file)).To(Succeed())
		out := make([]fuzzCase, 0, len(file.Cases))
		for _, vc := range file.Cases {
			enc := codec.NewStatic(vc.Keys, vc.DataTypes)
			dec := codec.NewStatic(vc.Keys, vc.DataTypes)
			fc := fuzzCase{Name: vc.Name}
			for _, k := range vc.Keys {
				fc.Keys = append(fc.Keys, uint32(k))
			}
			for _, dt := range vc.DataTypes {
				fc.DataTypes = append(fc.DataTypes, string(dt))
			}
			for _, f := range vc.Frames {
				ff := fuzzFrame{Name: f.Name, Series: f.Series}
				encoded, err := enc.Encode(ctx, multi(f.entries()...))
				if err != nil {
					ff.Error = err.Error()
					fc.Frames = append(fc.Frames, ff)
					continue
				}
				ff.Encoded = hex.EncodeToString(encoded)
				fr, err := dec.Decode(encoded)
				if err != nil {
					ff.Error = "decode: " + err.Error()
				} else {
					for k, s := range fr.Entries() {
						ff.Decoded = append(ff.Decoded, fuzzDecoded{
							Key:       uint32(k),
							DataType:  string(s.DataType),
							Data:      hex.EncodeToString(s.Data),
							Clock:     s.Clock,
							TimeRange: [2]int64{int64(s.TimeRange.Start), int64(s.TimeRange.End)},
							Alignment: uint64(s.Alignment),
						})
					}
				}
				fc.Frames = append(fc.Frames, ff)
			}
			out = append(out, fc)
		}
		b := MustSucceed(json.Marshal(map[string]any{"cases": out}))
		Expect(os.WriteFile(os.Getenv("FUZZ_OUT"), b, 0o644)).To(Succeed())
	})
})
