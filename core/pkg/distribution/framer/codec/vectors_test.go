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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// The vectors are shared with the TypeScript, Python, and C++ codecs, so that all four
// encode and decode each frame byte for byte.
const vectorsPath = "../../../../../x/testdata/framecodec/vectors.json"

type vectorFile struct {
	Cases []vectorCase `json:"cases"`
}

type vectorCase struct {
	Name      string           `json:"name"`
	Keys      channel.Keys     `json:"keys"`
	DataTypes []telem.DataType `json:"data_types"`
	Frames    []vectorFrame    `json:"frames"`
}

type vectorFrame struct {
	Name    string         `json:"name"`
	Series  []vectorSeries `json:"series"`
	Encoded string         `json:"encoded"`
}

type vectorSeries struct {
	Key      channel.Key    `json:"key"`
	DataType telem.DataType `json:"data_type"`
	// DecodedDataType is the type the decoder gives the series, when it differs from
	// DataType.
	DecodedDataType telem.DataType `json:"decoded_data_type"`
	Data            string         `json:"data"`
	Clock           uint32         `json:"clock"`
	// TimeRange holds the start and end of the series in nanoseconds.
	TimeRange [2]int64 `json:"time_range"`
	Alignment uint64   `json:"alignment"`
}

func (s vectorSeries) series(dt telem.DataType) telem.Series {
	return telem.Series{
		DataType: dt,
		Data:     MustSucceed(hex.DecodeString(s.Data)),
		Clock:    s.Clock,
		TimeRange: telem.TimeRange{
			Start: telem.TimeStamp(s.TimeRange[0]),
			End:   telem.TimeStamp(s.TimeRange[1]),
		},
		Alignment: telem.Alignment(s.Alignment),
	}
}

// entries returns the series that the frame encodes.
func (f vectorFrame) entries() []keyed {
	entries := make([]keyed, len(f.Series))
	for i, s := range f.Series {
		entries[i] = keyed{s.Key, s.series(s.DataType)}
	}
	return entries
}

// decoded returns the series that the frame decodes to.
func (f vectorFrame) decoded() []keyed {
	entries := make([]keyed, len(f.Series))
	for i, s := range f.Series {
		dt := s.DataType
		if s.DecodedDataType != "" {
			dt = s.DecodedDataType
		}
		entries[i] = keyed{s.Key, s.series(dt)}
	}
	return entries
}

var _ = Describe("Golden frames", func() {
	var file vectorFile
	Expect(json.Unmarshal(MustSucceed(os.ReadFile(vectorsPath)), &file)).To(Succeed())
	Expect(file.Cases).ToNot(BeEmpty())
	for _, vc := range file.Cases {
		It("Should encode and decode "+vc.Name+" byte for byte", func(ctx SpecContext) {
			enc := codec.NewStatic(vc.Keys, vc.DataTypes)
			dec := codec.NewStatic(vc.Keys, vc.DataTypes)
			for _, f := range vc.Frames {
				By(f.Name)
				encoded := MustSucceed(enc.Encode(ctx, multi(f.entries()...)))
				Expect(hex.EncodeToString(encoded)).To(Equal(f.Encoded))
				expectSeries(MustSucceed(dec.Decode(encoded)), f.decoded()...)
			}
		})
	}
})
