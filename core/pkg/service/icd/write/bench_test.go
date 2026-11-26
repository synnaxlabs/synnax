// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package write_test

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/write"
	"github.com/synnaxlabs/x/telem"
)

// benchDocuments returns the Documents of a message with a float field for each of n
// command channels, and a frame of samples commands to each channel.
func benchDocuments(b *testing.B, n, samples int) (*write.Documents, frame.Frame) {
	b.Helper()
	var (
		fields   = make([]icd.JSONField, 0, n)
		bindings = make([]icd.WriteBinding, 0, n)
		keys     = make(channel.Keys, n)
		series   = make([]telem.Series, n)
		channels = make(map[channel.Key]channel.Channel, n)
	)
	for i := range n {
		f := jsonField("f"+strconv.Itoa(i), icd.FloatJSONField{})
		fields = append(fields, f)
		keys[i] = channel.Key(i + 1)
		channels[keys[i]] = channel.Channel{DataType: telem.Float64T}
		bindings = append(bindings, channelOf(f.Base().Key, keys[i], nil))
		values := make([]float64, samples)
		for j := range values {
			values[j] = float64(i*samples+j) + 0.5
		}
		series[i] = telem.NewSeries(values)
	}
	m := jsonMessage("bench", fields...)
	ic := icd.ICD{Name: "bench", Messages: []icd.Message{m}}
	d, err := write.NewDocuments(writeOf(m, bindings...), ic, channels, "target")
	if err != nil {
		b.Fatal(err)
	}
	return d, frame.NewMulti(keys, series)
}

// checkDocuments checks that d builds the documents of benchDocuments from fr: one for
// each row, with the command of channel i at /fi.
func checkDocuments(b *testing.B, d *write.Documents, fr frame.Frame, n, samples int) {
	b.Helper()
	docs := 0
	for row, doc := range d.From(fr, driver.NewSkips("command", driver.SkipHold)) {
		want := []byte{'{'}
		for i := range n {
			if i > 0 {
				want = append(want, ',')
			}
			want = fmt.Appendf(want, "%q:", "f"+strconv.Itoa(i))
			want = strconv.AppendFloat(want, float64(i*samples+row)+0.5, 'g', -1, 64)
		}
		want = append(want, '}')
		if row != docs || !bytes.Equal(doc, want) {
			b.Fatalf("document %d is row %d, %s, not %s", docs, row, doc, want)
		}
		docs++
	}
	if docs != samples {
		b.Fatalf("built %d documents, want %d", docs, samples)
	}
}

func BenchmarkDocumentsFrom(b *testing.B) {
	for _, sc := range []struct{ channels, samples int }{
		{channels: 1, samples: 1},
		{channels: 1, samples: 100},
		{channels: 10, samples: 10},
		{channels: 100, samples: 10},
	} {
		name := fmt.Sprintf("channels=%d/samples=%d", sc.channels, sc.samples)
		b.Run(name, func(b *testing.B) {
			d, fr := benchDocuments(b, sc.channels, sc.samples)
			skips := driver.NewSkips("command", driver.SkipHold)
			checkDocuments(b, d, fr, sc.channels, sc.samples)
			b.ReportAllocs()
			for b.Loop() {
				docs := 0
				for range d.From(fr, skips) {
					docs++
				}
				if docs != sc.samples {
					b.Fatalf("built %d documents, want %d", docs, sc.samples)
				}
			}
			b.ReportMetric(
				float64(b.Elapsed().Nanoseconds())/float64(b.N*sc.samples), "ns/doc",
			)
		})
	}
}
