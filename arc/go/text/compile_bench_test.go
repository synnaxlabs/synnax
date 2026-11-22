// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package text_test

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// benchResolver finds a channel by name in one map probe, so the cost of a lookup does
// not grow with the number of channels. A miss returns one shared error, as the Core's
// consistent resolver answers a name it already looked up.
type benchResolver map[string]symbol.Symbol

// errBenchNotFound is the error of each lookup that benchResolver misses.
var errBenchNotFound = errors.Wrap(query.ErrNotFound, "channel")

func (r benchResolver) Resolve(_ context.Context, name string) (*symbol.Symbol, error) {
	if sym, ok := r[name]; ok {
		return &sym, nil
	}
	return nil, errBenchNotFound
}

func (r benchResolver) ResolveKey(context.Context, uint32) (*symbol.Symbol, error) {
	return nil, errBenchNotFound
}

func (r benchResolver) Search(context.Context, string) ([]*symbol.Symbol, error) {
	return nil, nil
}

// benchChannels returns a resolver that holds the sensor and output channels of n
// statements, each under the names that prefix gives.
func benchChannels(n int, prefix string) benchResolver {
	r := make(benchResolver, 2*n)
	for i := range n {
		for j, base := range []string{"sensor_", "out_"} {
			name := prefix + base + strconv.Itoa(i)
			r[name] = symbol.Symbol{
				Name: name,
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F64()),
				ID:   2*i + j + 1,
			}
		}
	}
	return r
}

// benchSource returns n statements made from the format of one, which takes the index
// of the statement twice.
func benchSource(header, format string, n int) string {
	var b strings.Builder
	b.WriteString(header)
	for i := range n {
		fmt.Fprintf(&b, format, i, i)
	}
	return b.String()
}

func benchCompile(b *testing.B, source string, channels benchResolver) {
	b.Helper()
	ctx := context.Background()
	compile := func() error {
		parsed, diag := text.Parse(text.Text{Raw: source})
		if diag != nil {
			return errors.New(diag.String())
		}
		root := symbol.NewRoot(channels, stl.NewSymbols())
		inter, diag := text.Analyze(ctx, parsed, root)
		if !diag.Ok() {
			return errors.New(diag.String())
		}
		_, err := text.Compile(ctx, inter)
		return err
	}
	if err := compile(); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if err := compile(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCompile parses, analyzes, and compiles programs of n statements, each of
// which reads one channel and writes another, against a resolver of one map probe.
func BenchmarkCompile(b *testing.B) {
	for _, n := range []int{10, 100} {
		suffix := "=" + strconv.Itoa(n)
		b.Run("names"+suffix, func(b *testing.B) {
			benchCompile(
				b,
				benchSource("", "sensor_%d * 2.0 -> out_%d\n", n),
				benchChannels(n, ""),
			)
		})
		b.Run("bare"+suffix, func(b *testing.B) {
			benchCompile(
				b,
				benchSource("", "stand_a.sensor_%d * 2.0 -> stand_a.out_%d\n", n),
				benchChannels(n, "stand_a."),
			)
		})
		b.Run("calls"+suffix, func(b *testing.B) {
			benchCompile(
				b,
				benchSource(
					"import math\n\n",
					"sensor_%d -> math.avg() -> out_%d\n",
					n,
				),
				benchChannels(n, ""),
			)
		})
		b.Run("constants"+suffix, func(b *testing.B) {
			benchCompile(
				b,
				benchSource("import math\n\n", "sensor_%d + math.inf -> out_%d\n", n),
				benchChannels(n, ""),
			)
		})
	}
}
