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
	"bytes"
	"compress/flate"
	"context"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"os"
	"sync"
	"testing"
	"text/tabwriter"
	"time"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/x/errors"
	xhttp "github.com/synnaxlabs/x/http"
	"github.com/synnaxlabs/x/telem"
)

// streamCompressions are the encodings a websocket peer could negotiate, in the order
// the HTTP transport prefers them.
var streamCompressions = []xhttp.Compression{xhttp.Zstd, xhttp.Brotli, xhttp.Gzip}

// streamPayload is one encoded frame under measurement, alongside the shape of signal
// that produced it.
type streamPayload struct {
	// name identifies the payload in the report.
	name string
	// encoded is the frame as the streaming codec puts it on the wire.
	encoded []byte
}

// streamCorpus encodes a set of frames with the real streaming codec, so the measured
// ratios describe the bytes that actually travel over a websocket rather than a
// hand-built approximation. Shapes run from a monotonic index channel, which is nearly
// pure redundancy, to full-entropy noise, which is the floor for any general-purpose
// encoding.
func streamCorpus(ctx context.Context, t testing.TB) []streamPayload {
	t.Helper()
	const samples = 5000
	// A fixed seed keeps the corpus identical across runs so comparisons are not
	// confounded by different data.
	r := rand.New(rand.NewPCG(42, 42))
	cases := []struct {
		name     string
		dataType telem.DataType
		series   func() telem.Series
	}{
		{"index-timestamps", telem.TimeStampT, func() telem.Series {
			// A 1kHz index channel: monotonic nanosecond timestamps at a fixed rate.
			out := make([]int64, samples)
			start := int64(1740000000000000000)
			for i := range out {
				out[i] = start + int64(i)*1_000_000
			}
			return telem.NewSeriesV(out...)
		}},
		{"float64-sine", telem.Float64T, func() telem.Series {
			out := make([]float64, samples)
			for i := range out {
				out[i] = math.Sin(float64(i) / 50)
			}
			return telem.NewSeriesV(out...)
		}},
		{"float64-steady", telem.Float64T, func() telem.Series {
			// A sensor parked at a setpoint with a small amount of jitter, which is
			// what most control-loop channels look like between events.
			out := make([]float64, samples)
			for i := range out {
				out[i] = 101.325 + math.Round(r.NormFloat64()*10)/1000
			}
			return telem.NewSeriesV(out...)
		}},
		{"float64-noise", telem.Float64T, func() telem.Series {
			out := make([]float64, samples)
			for i := range out {
				out[i] = r.Float64() * 1e6
			}
			return telem.NewSeriesV(out...)
		}},
		{"float32-sine", telem.Float32T, func() telem.Series {
			out := make([]float32, samples)
			for i := range out {
				out[i] = float32(math.Sin(float64(i) / 50))
			}
			return telem.NewSeriesV(out...)
		}},
		{"uint8-state", telem.Uint8T, func() telem.Series {
			// A discrete state channel that rarely changes: valve open/closed.
			out := make([]uint8, samples)
			for i := range out {
				out[i] = uint8((i / 500) % 2)
			}
			return telem.NewSeriesV(out...)
		}},
	}
	payloads := make([]streamPayload, 0, len(cases)+1)
	var (
		mixedKeys   channel.Keys
		mixedSeries []telem.Series
		mixedTypes  []telem.DataType
	)
	for i, c := range cases {
		key := channel.Key(i + 1)
		series := c.series()
		cdc := codec.NewStatic(channel.Keys{key}, []telem.DataType{c.dataType})
		encoded, err := cdc.Encode(ctx, frame.NewMulti(
			channel.Keys{key}, []telem.Series{series},
		))
		if err != nil {
			t.Fatal(err)
		}
		payloads = append(payloads, streamPayload{c.name, encoded})
		mixedKeys = append(mixedKeys, key)
		mixedSeries = append(mixedSeries, series)
		mixedTypes = append(mixedTypes, c.dataType)
	}
	// One frame carrying every channel at once, which is the shape a Console dashboard
	// actually subscribes to.
	cdc := codec.NewStatic(mixedKeys, mixedTypes)
	encoded, err := cdc.Encode(ctx, frame.NewMulti(mixedKeys, mixedSeries))
	if err != nil {
		t.Fatal(err)
	}
	return append(payloads, streamPayload{"mixed-frame", encoded})
}

// TestStreamFrameCompression reports how far each encoding shrinks a frame already
// packed by the streaming codec, and the link speed at which that shrinking repays its
// CPU cost. The unary HTTP path compresses JSON, which is mostly punctuation and
// decimal text; this path compresses little-endian sample data, so the ratios here are
// the ones that decide whether websocket compression is worth enabling.
//
//	go test ./pkg/distribution/framer/codec/ -run TestStreamFrameCompression -v
func TestStreamFrameCompression(t *testing.T) {
	ctx := context.Background()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	var writeErr error
	row := func(format string, args ...any) {
		if writeErr == nil {
			_, writeErr = fmt.Fprintf(w, format+"\n", args...)
		}
	}
	row("payload\tencoding\tencoded\tcompressed\tratio\tcompress\tbreak-even")
	for _, p := range streamCorpus(ctx, t) {
		for _, c := range streamCompressions {
			out, err := c.Compress(p.encoded)
			if err != nil {
				t.Fatal(err)
			}
			compress := timeCompression(t, func() error {
				_, err := c.Compress(p.encoded)
				return err
			})
			decompress := timeCompression(t, func() error {
				_, err := c.Decompress(out, 0)
				return err
			})
			saved := len(p.encoded) - len(out)
			breakEven := float64(saved) / (compress + decompress).Seconds() * 8 / 1e6
			row(
				"%s\t%s\t%d\t%d\t%.3f\t%v\t%.0f Mbps",
				p.name, c.ContentEncoding(), len(p.encoded), len(out),
				float64(len(out))/float64(len(p.encoded)),
				compress.Round(time.Microsecond),
				breakEven,
			)
		}
	}
	if err := errors.Combine(writeErr, w.Flush()); err != nil {
		t.Fatal(err)
	}
}

// deflateWriters mirrors the pool the websocket library keeps, so the measured cost is
// per message rather than per writer allocation.
var deflateWriters = sync.Pool{New: func() any {
	w, err := flate.NewWriter(io.Discard, 1)
	if err != nil {
		panic(err)
	}
	return w
}}

// permessageDeflate compresses src the way a websocket negotiating permessage-deflate
// does: raw flate at level 1, with a window that is not carried between messages. The
// four-byte empty block that terminates a message is dropped by the protocol, so it is
// subtracted here.
func permessageDeflate(src []byte) ([]byte, error) {
	w := deflateWriters.Get().(*flate.Writer)
	defer deflateWriters.Put(w)
	var buf bytes.Buffer
	w.Reset(&buf)
	if _, err := w.Write(src); err != nil {
		return nil, err
	}
	if err := w.Flush(); err != nil {
		return nil, err
	}
	out := buf.Bytes()
	if len(out) >= 4 {
		out = out[:len(out)-4]
	}
	return out, nil
}

// TestStreamFrameSmall reports what permessage-deflate does to the small frames a live
// subscription actually produces. The extension has no minimum size, so every data
// message is deflated however small, and no dictionary carries between messages: each
// frame is compressed alone, against nothing.
//
//	go test ./pkg/distribution/framer/codec/ -run TestStreamFrameSmall -v
func TestStreamFrameSmall(t *testing.T) {
	ctx := context.Background()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	var writeErr error
	row := func(format string, args ...any) {
		if writeErr == nil {
			_, writeErr = fmt.Fprintf(w, format+"\n", args...)
		}
	}
	row("channels\tsamples\tencoded\tdeflated\tdelta\tratio\tdeflate")
	// A live Console subscription receives a frame per acquisition tick. One sample
	// per channel is the high-rate limit; the larger counts cover slower consumers
	// that batch several ticks together.
	for _, shape := range []struct{ channels, samples int }{
		{1, 1}, {8, 1}, {32, 1}, {8, 10}, {32, 10}, {8, 100}, {32, 100},
	} {
		// Each channel gets its own baseline and its own jitter, so the frame is not
		// N copies of one series. Identical channels would make deflate look far
		// better here than it does against a real rack.
		r := rand.New(rand.NewPCG(42, uint64(shape.channels)))
		keys := make(channel.Keys, shape.channels)
		types := make([]telem.DataType, shape.channels)
		series := make([]telem.Series, shape.channels)
		for i := range shape.channels {
			keys[i] = channel.Key(i + 1)
			types[i] = telem.Float64T
			values := make([]float64, shape.samples)
			base := 10 + float64(i)*13.7
			for j := range values {
				values[j] = base + math.Round(r.NormFloat64()*1000)/1000
			}
			series[i] = telem.NewSeriesV(values...)
		}
		cdc := codec.NewStatic(keys, types)
		encoded, err := cdc.Encode(ctx, frame.NewMulti(keys, series))
		if err != nil {
			t.Fatal(err)
		}
		deflated, err := permessageDeflate(encoded)
		if err != nil {
			t.Fatal(err)
		}
		elapsed := timeCompression(t, func() error {
			_, err := permessageDeflate(encoded)
			return err
		})
		row(
			"%d\t%d\t%d\t%d\t%+d\t%.3f\t%v",
			shape.channels, shape.samples, len(encoded), len(deflated),
			len(deflated)-len(encoded),
			float64(len(deflated))/float64(len(encoded)),
			elapsed.Round(time.Microsecond),
		)
	}
	if err := errors.Combine(writeErr, w.Flush()); err != nil {
		t.Fatal(err)
	}
}

// shuffle regroups src so that every element's first byte is contiguous, then every
// element's second byte, and so on for a fixed element width. Byte-oriented encodings
// see an IEEE-754 array as interleaved sign, exponent, and mantissa bytes and find
// almost no redundancy; after transposition the exponent bytes of a bounded signal
// become long runs while the noisy low mantissa bytes group together. Any trailing
// bytes that do not fill a whole element are copied through.
func shuffle(src []byte, width int) []byte {
	n := len(src) / width
	out := make([]byte, len(src))
	for i := range n {
		for j := range width {
			out[j*n+i] = src[i*width+j]
		}
	}
	copy(out[n*width:], src[n*width:])
	return out
}

// TestStreamFrameShuffle reports what byte transposition adds on top of a general
// purpose encoding. It answers whether a Synnax-specific step ahead of the encoder is
// worth building, or whether negotiating permessage-deflate on the websocket is the
// whole of the available win.
//
//	go test ./pkg/distribution/framer/codec/ -run TestStreamFrameShuffle -v
func TestStreamFrameShuffle(t *testing.T) {
	ctx := context.Background()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	var writeErr error
	row := func(format string, args ...any) {
		if writeErr == nil {
			_, writeErr = fmt.Fprintf(w, format+"\n", args...)
		}
	}
	row("payload\twidth\tencoded\tzstd\tshuffle+zstd\tratio\tshuffle")
	// Widths match the sample size of each corpus entry; the mixed frame holds several
	// widths at once, so transposing it as a whole is not meaningful.
	widths := map[string]int{
		"index-timestamps": 8,
		"float64-sine":     8,
		"float64-steady":   8,
		"float64-noise":    8,
		"float32-sine":     4,
		"uint8-state":      1,
	}
	// perSeriesBest sums the smaller of the two options for each channel, modelling a
	// codec that decides per series instead of compressing the whole frame as one blob.
	var perSeriesRaw, perSeriesBest int
	corpus := streamCorpus(ctx, t)
	for _, p := range corpus {
		width, ok := widths[p.name]
		if !ok {
			continue
		}
		plain, err := xhttp.Zstd.Compress(p.encoded)
		if err != nil {
			t.Fatal(err)
		}
		shuffled, err := xhttp.Zstd.Compress(shuffle(p.encoded, width))
		if err != nil {
			t.Fatal(err)
		}
		shuffleTime := timeCompression(t, func() error {
			shuffle(p.encoded, width)
			return nil
		})
		perSeriesRaw += len(p.encoded)
		perSeriesBest += min(len(plain), len(shuffled))
		row(
			"%s\t%d\t%d\t%d\t%d\t%.3f\t%v",
			p.name, width, len(p.encoded), len(plain), len(shuffled),
			float64(len(shuffled))/float64(len(p.encoded)),
			shuffleTime.Round(time.Microsecond),
		)
	}
	if err := errors.Combine(writeErr, w.Flush()); err != nil {
		t.Fatal(err)
	}
	// The whole-frame figures the per-series total has to beat to justify the extra
	// machinery.
	for _, p := range corpus {
		if p.name != "mixed-frame" {
			continue
		}
		wholeZstd, err := xhttp.Zstd.Compress(p.encoded)
		if err != nil {
			t.Fatal(err)
		}
		wholeGzip, err := xhttp.Gzip.Compress(p.encoded)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf(
			"mixed frame %d bytes: whole-frame gzip %d (%.3f), whole-frame zstd %d "+
				"(%.3f), per-series best-of-two %d (%.3f)",
			len(p.encoded),
			len(wholeGzip), float64(len(wholeGzip))/float64(len(p.encoded)),
			len(wholeZstd), float64(len(wholeZstd))/float64(len(p.encoded)),
			perSeriesBest, float64(perSeriesBest)/float64(perSeriesRaw),
		)
	}
}

// timeCompression returns the average wall time of fn over enough iterations to outrun
// timer granularity.
func timeCompression(t testing.TB, fn func() error) time.Duration {
	t.Helper()
	const iterations = 100
	// One warm-up run pulls the pooled writer and its buffers into existence so the
	// first timed iteration is not charged for them.
	if err := fn(); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	for range iterations {
		if err := fn(); err != nil {
			t.Fatal(err)
		}
	}
	return time.Since(start) / iterations
}
