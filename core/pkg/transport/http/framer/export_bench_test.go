// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer_test

import (
	"context"
	"io"
	"testing"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apiframer "github.com/synnaxlabs/synnax/pkg/api/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	svcchannel "github.com/synnaxlabs/synnax/pkg/service/channel"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/transport/http/framer"
	xhttp "github.com/synnaxlabs/x/http"
	"github.com/synnaxlabs/x/telem"
)

// openExport returns a read of samples float64 samples of one channel and its index,
// one second apart, and a function that closes its Core.
func openExport(b *testing.B, samples int) (apiframer.ReadResponse, func()) {
	b.Helper()
	gomega.RegisterTestingT(b)
	ctx := b.Context()
	node := testutil.OpenNode(ctx)
	layer, err := svctestutil.OpenLayer(ctx, node)
	if err != nil {
		b.Fatal(err)
	}
	index := svcchannel.Channel{Name: "time", DataType: telem.TimestampT, IsIndex: true}
	if err = node.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return layer.Channel.NewWriter(tx).Create(ctx, &index)
	}); err != nil {
		b.Fatal(err)
	}
	data := svcchannel.Channel{
		Name:       "value",
		DataType:   telem.Float64T,
		LocalIndex: index.LocalKey,
	}
	if err = node.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return layer.Channel.NewWriter(tx).Create(ctx, &data)
	}); err != nil {
		b.Fatal(err)
	}
	stamps := make([]telem.TimeStamp, samples)
	values := make([]float64, samples)
	for i := range stamps {
		stamps[i] = telem.TimeStamp(i+1) * telem.SecondTS
	}
	keys := svcchannel.Keys{index.Key(), data.Key()}
	w, err := layer.Framer.OpenWriter(ctx, svcframer.WriterConfig{
		Start: stamps[0],
		Keys:  keys,
	})
	if err != nil {
		b.Fatal(err)
	}
	authorized, err := w.Write(frame.NewMulti(keys, []telem.Series{
		telem.NewSeries(stamps),
		telem.NewSeries(values),
	}))
	if err != nil || !authorized {
		b.Fatalf("write failed: %v", err)
	}
	if err = w.Close(); err != nil {
		b.Fatal(err)
	}
	res := apiframer.ReadResponse{
		Channels: []svcchannel.Channel{data},
		Indexes:  []svcchannel.Channel{index},
		Bounds:   telem.TimeRangeMax,
		OpenIterator: func(
			ctx context.Context,
			keys svcchannel.Keys,
			bounds telem.TimeRange,
		) (*svcframer.Iterator, error) {
			return layer.Framer.OpenIterator(ctx, svcframer.IteratorConfig{
				Keys:   keys,
				Bounds: bounds,
			})
		},
	}
	return res, func() {
		if err := layer.Close(); err != nil {
			b.Error(err)
		}
		if err := node.Close(); err != nil {
			b.Error(err)
		}
	}
}

// BenchmarkExport encodes a read of 100,000 samples of one channel and its index.
func BenchmarkExport(b *testing.B) {
	for _, enc := range []struct {
		name string
		open func(context.Context) xhttp.Encoder
	}{
		{"CSV", framer.NewCSVEncoder},
		{"TDMS", framer.NewTDMSEncoder},
	} {
		b.Run(enc.name, func(b *testing.B) {
			res, closeCore := openExport(b, 100_000)
			defer closeCore()
			b.ReportAllocs()
			for b.Loop() {
				if err := enc.open(b.Context()).
					EncodeStream(io.Discard, res); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
