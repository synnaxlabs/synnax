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
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
)

// payload is one entry in the benchmark corpus: a JSON body shaped like something the
// Synnax API actually puts on the wire.
type payload struct {
	// name identifies the payload in benchmark output.
	name string
	// body is the encoded JSON body.
	body []byte
}

// corpus returns the benchmark payloads ordered from smallest to largest. The shapes
// mirror real API responses so the measured ratios carry over to production traffic:
// channel metadata dominates the small end, and telemetry frames dominate the large
// end.
func corpus() []payload {
	return []payload{
		{"auth-response", mustEncode(authResponse())},
		{"channel-retrieve-1", mustEncode(channelResponse(1))},
		{"channel-retrieve-10", mustEncode(channelResponse(10))},
		{"channel-retrieve-100", mustEncode(channelResponse(100))},
		{"channel-retrieve-1000", mustEncode(channelResponse(1000))},
		{"ontology-tree-500", mustEncode(ontologyResponse(500))},
		{"frame-sine-8x1000", mustEncode(frameResponse(8, 1000, sine))},
		{"frame-noise-8x1000", mustEncode(frameResponse(8, 1000, noise))},
		{"frame-sine-32x5000", mustEncode(frameResponse(32, 5000, sine))},
	}
}

func mustEncode(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// authResponse mirrors the login response: a JWT plus a small user record. It stands in
// for the many API responses that are a single short object.
func authResponse() any {
	return map[string]any{
		"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9." +
			strings.Repeat("aGVsbG8gc3lubmF4IHRva2Vu", 6) + ".c2lnbmF0dXJl",
		"user": map[string]any{
			"key":        "6f1b0c3a-2e4d-4f1a-9b7c-8d5e2f0a1b3c",
			"username":   "synnax",
			"first_name": "Syn",
			"last_name":  "Nax",
			"root_user":  true,
		},
	}
}

// channelResponse mirrors a channel retrieve response holding n channels. Channel
// metadata is highly repetitive across records, which is where compression pays best.
func channelResponse(n int) any {
	channels := make([]map[string]any, n)
	dataTypes := []string{"float32", "float64", "int32", "uint8", "timestamp"}
	for i := range channels {
		channels[i] = map[string]any{
			"key":         65536 + i,
			"name":        fmt.Sprintf("gse_pressure_transducer_%d", i),
			"leaseholder": 1,
			"data_type":   dataTypes[i%len(dataTypes)],
			"is_index":    false,
			"index":       65536,
			"virtual":     false,
			"internal":    false,
			"expression":  "",
			"concurrency": 0,
		}
	}
	return map[string]any{"channels": channels}
}

// ontologyResponse mirrors an ontology resource retrieve, whose records carry nested
// IDs and a name — the shape behind the Console's resource tree.
func ontologyResponse(n int) any {
	resources := make([]map[string]any, n)
	types := []string{"channel", "range", "rack", "device", "schematic"}
	for i := range resources {
		t := types[i%len(types)]
		resources[i] = map[string]any{
			"id": map[string]any{
				"type": t,
				"key":  fmt.Sprintf("%s-%d", t, i),
			},
			"name": fmt.Sprintf("%s %d", t, i),
			"data": map[string]any{
				"key":    fmt.Sprintf("%s-%d", t, i),
				"name":   fmt.Sprintf("%s %d", t, i),
				"parent": "root",
			},
		}
	}
	return map[string]any{"resources": resources}
}

// series generates the sample values for one channel of a frame.
type series func(r *rand.Rand, n int) []float64

// sine produces a smooth waveform, standing in for a well-behaved sensor. Its digits
// repeat heavily once rounded, so it represents the compressible end of telemetry.
func sine(_ *rand.Rand, n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = math.Round(math.Sin(float64(i)/50)*1000) / 1000
	}
	return out
}

// noise produces uniform random values, the worst realistic case for telemetry: full
// mantissa entropy with nothing for a dictionary to reuse.
func noise(r *rand.Rand, n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = math.Round(r.Float64()*1e6) / 1000
	}
	return out
}

// frameResponse mirrors a framer read response: channel keys alongside one series of
// samples each. JSON renders every sample as decimal text, which is why frames are the
// heaviest bodies the HTTP API returns.
func frameResponse(channels, samples int, gen series) any {
	// A fixed seed keeps the corpus identical across runs so benchmark comparisons
	// are not confounded by different data.
	r := rand.New(rand.NewPCG(42, 42))
	keys := make([]int, channels)
	all := make([]map[string]any, channels)
	for i := range channels {
		keys[i] = 65536 + i
		all[i] = map[string]any{
			"data_type": "float64",
			"time_range": map[string]any{
				"start": 1740000000000000000,
				"end":   1740000000000000000 + samples*1000000,
			},
			"alignment": 0,
			"data":      gen(r, samples),
		}
	}
	return map[string]any{"keys": keys, "series": all}
}
