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
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	fhttp "github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/freighter/test"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/encoding/json"
	xhttp "github.com/synnaxlabs/x/http"
	xnet "github.com/synnaxlabs/x/net"
)

// benchPayload is one request/response size under benchmark. Message is echoed back by
// the server, so it sets both the request and the response body size.
type benchPayload struct {
	name    string
	message string
}

// benchPayloads spans the range of API bodies, from a single-record response up to a
// bulk channel listing.
func benchPayloads() []benchPayload {
	return []benchPayload{
		{"256B", channelRecords(256)},
		{"4KB", channelRecords(4096)},
		{"64KB", channelRecords(65536)},
		{"1MB", channelRecords(1048576)},
	}
}

// channelRecords returns a message of exactly n bytes shaped like a channel listing.
// Keys and names differ per record, so the result carries the structural redundancy of
// real metadata without the perfect self-similarity of a single repeated string, which
// would flatter every encoding.
func channelRecords(n int) string {
	var b strings.Builder
	b.Grow(n + 128)
	dataTypes := []string{"float32", "float64", "int32", "uint8", "timestamp"}
	for i := 0; b.Len() < n; i++ {
		fmt.Fprintf(
			&b,
			`{"key":%d,"name":"gse_pressure_transducer_%d","data_type":"%s",`+
				`"is_index":false,"index":65536,"virtual":false},`,
			65536+i, i, dataTypes[i%len(dataTypes)],
		)
	}
	return b.String()[:n]
}

// benchServer starts a unary echo server on a free port and returns its address. The
// server is shut down when the benchmark finishes.
func benchServer(tb testing.TB, opts ...fhttp.UnaryServerOption) address.Address {
	tb.Helper()
	port, err := xnet.FindOpenPort()
	if err != nil {
		tb.Fatal(err)
	}
	addr := address.Newf("localhost:%d", port)
	app := fiber.New(fiber.Config{})
	app.Server().MaxIdleWorkerDuration = 100 * time.Millisecond
	router, err := fhttp.NewRouter()
	if err != nil {
		tb.Fatal(err)
	}
	app.Get("/health", func(ctx fiber.Ctx) error {
		return ctx.SendStatus(fiber.StatusOK)
	})
	server := fhttp.NewUnaryServer[test.Request, test.Response](router, "/", opts...)
	server.BindHandler(
		func(_ context.Context, req test.Request) (test.Response, error) {
			return test.Response(req), nil
		},
	)
	router.BindTo(app)
	go func() {
		_ = app.Listen(addr.PortString(), fiber.ListenConfig{
			DisableStartupMessage: true,
		})
	}()
	waitHealthy(tb, "http://"+addr.String()+"/health")
	tb.Cleanup(func() {
		if err := app.Shutdown(); err != nil {
			tb.Error(err)
		}
	})
	return addr
}

func waitHealthy(tb testing.TB, url string) {
	tb.Helper()
	for range 500 {
		res, err := http.Get(url)
		if err == nil {
			if err := res.Body.Close(); err != nil {
				tb.Fatal(err)
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
	tb.Fatalf("server at %s never became healthy", url)
}

// BenchmarkUnaryRoundTrip measures a full client-to-server-and-back request over
// loopback, with and without compression.
//
// Two things bound what this can show. Loopback has effectively unlimited bandwidth, so
// the benchmark charges the full CPU cost of compression and credits none of the
// transmission time it saves. The unary client also closes its connection after every
// response, so a large share of each figure is TCP setup rather than body handling.
// Read it as an upper bound on the cost, not as the whole trade: the break-even test in
// x/http reports the link speed at which that cost is repaid.
func BenchmarkUnaryRoundTrip(b *testing.B) {
	addr := benchServer(b)
	configs := []struct {
		name string
		cfg  fhttp.UnaryClientConfig
	}{
		{"none", fhttp.UnaryClientConfig{DisableCompression: true}},
		{"zstd", fhttp.UnaryClientConfig{
			Compressions: []xhttp.Compression{xhttp.Zstd}, MinCompressSize: 1,
		}},
		{"br", fhttp.UnaryClientConfig{
			Compressions: []xhttp.Compression{xhttp.Brotli}, MinCompressSize: 1,
		}},
		{"gzip", fhttp.UnaryClientConfig{
			Compressions: []xhttp.Compression{xhttp.Gzip}, MinCompressSize: 1,
		}},
	}
	for _, p := range benchPayloads() {
		for _, c := range configs {
			b.Run(p.name+"/"+c.name, func(b *testing.B) {
				client, err := fhttp.NewUnaryClient[test.Request, test.Response](c.cfg)
				if err != nil {
					b.Fatal(err)
				}
				req := test.Request{ID: 1, Message: p.message}
				b.SetBytes(int64(len(p.message)))
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					if _, err := client.Send(b.Context(), addr, req); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// TestUnaryWireSize prints the bytes actually put on the wire in each direction for
// every payload and encoding, measured against a live server. It is the end-to-end
// counterpart to the codec-level ratios: what a real request and its response cost
// after JSON encoding and HTTP framing.
//
//	go test ./http/ -run TestUnaryWireSize -v
func TestUnaryWireSize(t *testing.T) {
	addr := benchServer(t)
	encodings := []string{"", "zstd", "br", "gzip"}
	tbl := newTable("payload\tencoding\traw\treq_wire\tres_wire\tratio")
	for _, p := range benchPayloads() {
		req := test.Request{ID: 1, Message: p.message}
		raw, err := json.Codec.Encode(t.Context(), req)
		if err != nil {
			t.Fatal(err)
		}
		for _, encoding := range encodings {
			reqWire, resWire := measureWire(t, addr, raw, encoding)
			name := encoding
			if name == "" {
				name = "none"
			}
			tbl.row(
				"%s\t%s\t%d\t%d\t%d\t%.3f",
				p.name, name, len(raw), reqWire, resWire,
				float64(resWire)/float64(len(raw)),
			)
		}
	}
	tbl.flush(t)
}

// measureWire sends raw to the server under the given encoding and returns the request
// and response body sizes as they travelled. An empty encoding sends and accepts an
// uncompressed body.
func measureWire(
	t *testing.T,
	addr address.Address,
	raw []byte,
	encoding string,
) (int, int) {
	t.Helper()
	body := raw
	if encoding != "" {
		compression, err := xhttp.ResolveCompression(
			encoding, []xhttp.Compression{xhttp.Zstd, xhttp.Brotli, xhttp.Gzip},
		)
		if err != nil {
			t.Fatal(err)
		}
		if body, err = compression.Compress(raw); err != nil {
			t.Fatal(err)
		}
	}
	httpReq, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"http://"+addr.String()+"/",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatal(err)
	}
	httpReq.Header.Set(fiber.HeaderContentType, "application/json")
	httpReq.Header.Set(fiber.HeaderAccept, "application/json")
	if encoding != "" {
		httpReq.Header.Set(fiber.HeaderContentEncoding, encoding)
	}
	// Set explicitly so net/http does not add its own and transparently decompress,
	// which would report the decoded size rather than the transmitted one.
	httpReq.Header.Set(fiber.HeaderAcceptEncoding, encoding)
	httpRes, err := (&http.Client{}).Do(httpReq)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := httpRes.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	if httpRes.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status %d for encoding %q", httpRes.StatusCode, encoding)
	}
	resBody, err := io.ReadAll(httpRes.Body)
	if err != nil {
		t.Fatal(err)
	}
	return len(body), len(resBody)
}
