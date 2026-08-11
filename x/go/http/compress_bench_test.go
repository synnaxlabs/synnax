// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http_test

import (
	"testing"
	"time"

	xhttp "github.com/synnaxlabs/x/http"
)

// compressions is the set under benchmark, in the order the server offers them.
var compressions = []xhttp.Compression{xhttp.Zstd, xhttp.Brotli, xhttp.Gzip}

// BenchmarkCompress measures the cost of compressing each corpus payload. The reported
// ratio metric is compressed size over original size, so lower is better and a value
// above 1 means the encoding made the body bigger.
func BenchmarkCompress(b *testing.B) {
	for _, p := range corpus() {
		for _, c := range compressions {
			b.Run(p.name+"/"+c.ContentEncoding(), func(b *testing.B) {
				out, err := c.Compress(p.body)
				if err != nil {
					b.Fatal(err)
				}
				b.SetBytes(int64(len(p.body)))
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					if _, err := c.Compress(p.body); err != nil {
						b.Fatal(err)
					}
				}
				b.StopTimer()
				b.ReportMetric(float64(len(out))/float64(len(p.body)), "ratio")
				b.ReportMetric(float64(len(out)), "out_bytes")
			})
		}
	}
}

// BenchmarkDecompress measures the cost of decompressing each corpus payload, which is
// the work a client pays on every response.
func BenchmarkDecompress(b *testing.B) {
	for _, p := range corpus() {
		for _, c := range compressions {
			b.Run(p.name+"/"+c.ContentEncoding(), func(b *testing.B) {
				out, err := c.Compress(p.body)
				if err != nil {
					b.Fatal(err)
				}
				b.SetBytes(int64(len(p.body)))
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					if _, err := c.Decompress(out, 0); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// BenchmarkNegotiateCompression measures the per-request overhead of parsing an
// Accept-Encoding header, which every compressed response pays on top of the encoding
// itself.
func BenchmarkNegotiateCompression(b *testing.B) {
	for _, header := range []string{
		"gzip, deflate, br, zstd",
		"gzip;q=0.5, br;q=1.0, *;q=0",
		"identity",
	} {
		b.Run(header, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				xhttp.NegotiateCompression(header, compressions)
			}
		})
	}
}

// TestCompressionProfile prints the size and throughput of every encoding over every
// corpus payload as a single table. Benchmark output alone makes cross-payload
// comparison hard, and the size question — where compression starts paying for itself —
// is the one that decides the transport's minimum body size. Run it with:
//
//	go test ./http/ -run TestCompressionProfile -v
func TestCompressionProfile(t *testing.T) {
	tbl := newTable("payload\tencoding\traw\tencoded\tsaved\tratio")
	for _, p := range corpus() {
		for _, c := range compressions {
			out, err := c.Compress(p.body)
			if err != nil {
				t.Fatal(err)
			}
			tbl.row(
				"%s\t%s\t%d\t%d\t%+d\t%.3f",
				p.name,
				c.ContentEncoding(),
				len(p.body),
				len(out),
				len(p.body)-len(out),
				float64(len(out))/float64(len(p.body)),
			)
		}
	}
	tbl.flush(t)
}

// TestCompressionBreakEven prints, per payload and encoding, the link speed at which
// compression stops costing more than it saves.
//
// Compression is worth it when the CPU time it costs is less than the transmission time
// it removes, so the break-even link speed is bytes_saved / (compress + decompress
// time). On a link slower than that figure compression is a net win; on a faster one it
// is a net loss. Loopback and a same-host Console sit well above any of these numbers,
// while a 1 Gbps LAN, a VPN, or a browser over the internet sit below them.
//
//	go test ./http/ -run TestCompressionBreakEven -v
func TestCompressionBreakEven(t *testing.T) {
	tbl := newTable(
		"payload\tencoding\traw\tsaved\tcompress\tdecompress\tbreak-even",
	)
	for _, p := range corpus() {
		for _, c := range compressions {
			out, err := c.Compress(p.body)
			if err != nil {
				t.Fatal(err)
			}
			compress := timePer(func() {
				if _, err := c.Compress(p.body); err != nil {
					t.Fatal(err)
				}
			})
			decompress := timePer(func() {
				if _, err := c.Decompress(out, 0); err != nil {
					t.Fatal(err)
				}
			})
			saved := len(p.body) - len(out)
			// Bytes per second, converted to megabits per second.
			breakEven := float64(saved) / (compress + decompress).Seconds() * 8 / 1e6
			tbl.row(
				"%s\t%s\t%d\t%+d\t%v\t%v\t%.0f Mbps",
				p.name, c.ContentEncoding(), len(p.body), saved,
				compress.Round(time.Microsecond),
				decompress.Round(time.Microsecond),
				breakEven,
			)
		}
	}
	tbl.flush(t)
}

// timePer returns the average wall time of fn over enough iterations to outrun timer
// granularity.
func timePer(fn func()) time.Duration {
	const iterations = 200
	// One warm-up run pulls the pooled writer and its buffers into existence so the
	// first timed iteration is not charged for them.
	fn()
	start := time.Now()
	for range iterations {
		fn()
	}
	return time.Since(start) / iterations
}

// TestCompressionFloor prints the smallest body each encoding compresses without
// growing it. It answers whether the transport needs a minimum-size gate at all, and if
// so where to put it.
func TestCompressionFloor(t *testing.T) {
	tbl := newTable("size\tzstd\tbr\tgzip")
	// Sizes bracket the small end of the API, where most responses land.
	for _, size := range []int{16, 32, 64, 128, 200, 256, 384, 512, 1024, 2048} {
		body := mustEncode(channelResponse(1))
		for len(body) < size {
			body = append(body, mustEncode(channelResponse(1))...)
		}
		body = body[:size]
		deltas := make([]any, 0, len(compressions)+1)
		deltas = append(deltas, size)
		for _, c := range compressions {
			out, err := c.Compress(body)
			if err != nil {
				t.Fatal(err)
			}
			deltas = append(deltas, len(out)-len(body))
		}
		tbl.row("%d\t%+d\t%+d\t%+d", deltas...)
	}
	tbl.flush(t)
}
