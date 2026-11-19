// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ingest

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ingestv1 "github.com/sift-stack/sift/go/gen/sift/ingest/v1"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("encode", func() {
	// D-111, amended: a series on a clock other than its flow's index fails, and the
	// error names the channel and both clock keys.
	It(
		"Should fail on a series whose clock differs from the index of its flow",
		func() {
			index := channel.Channel{
				Name:        "time",
				Leaseholder: 1,
				LocalKey:    1,
				IsIndex:     true,
				DataType:    telem.TimestampT,
			}
			data := channel.Channel{
				Name:        "pressure",
				Leaseholder: 1,
				LocalKey:    2,
				LocalIndex:  1,
				DataType:    telem.Float32T,
			}
			stamps := telem.NewSeriesSecondsTSV(1)
			stamps.Clock = uint32(index.Key())
			samples := telem.NewSeriesV[float32](1)
			samples.Clock = 99
			fr := frame.NewMulti(
				[]channel.Key{index.Key(), data.Key()},
				[]telem.Series{stamps, samples},
			)
			_, _, err := encode(
				"config",
				flow{index: index, channels: []channel.Channel{data}},
				fr,
				0,
				[]telem.TimeStamp{0},
				func(*ingestv1.IngestWithConfigDataStreamRequest) error { return nil },
			)
			Expect(err).To(SatisfyAll(
				MatchError(ContainSubstring("pressure")),
				MatchError(ContainSubstring("99")),
				MatchError(
					ContainSubstring(strconv.FormatUint(uint64(index.Key()), 10)),
				),
			))
		},
	)
})
