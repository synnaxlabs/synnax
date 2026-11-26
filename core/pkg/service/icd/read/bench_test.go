// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package read_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/read"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
)

// benchSource returns a source of one message with a timestamp field at /ts and n
// float fields at /f0 to /f(n-1), and a document of it. A stamped source writes the
// floats to data channels of one index channel that /ts stamps. Else each float writes
// to a virtual channel of its own.
func benchSource(b *testing.B, n int, stamped bool) (*read.Source, []byte) {
	b.Helper()
	const leaseholder = 1
	ts := jsonField(
		"ts", "/ts", icd.TimestampJSONField{TimeFormat: payload.TimeFormatUnixNs},
	)
	fields := append(make([]icd.JSONField, 0, n+1), ts)
	channels := map[channel.Key]channel.Channel{}
	bindings := make([]icd.ReadBinding, 0, n+1)
	if stamped {
		index := channel.Channel{
			Leaseholder: leaseholder, LocalKey: 1, LocalIndex: 1, IsIndex: true,
			DataType: telem.TimestampT,
		}
		channels[index.Key()] = index
		bindings = append(bindings, bind(ts, index.Key()))
	}
	var doc strings.Builder
	doc.WriteString(`{"ts":1767323045123456789`)
	for i := range n {
		name := "f" + strconv.Itoa(i)
		f := jsonField(name, "/"+name, icd.FloatJSONField{})
		fields = append(fields, f)
		ch := channel.Channel{
			Leaseholder: leaseholder,
			LocalKey:    channel.LocalKey(i + 2),
			DataType:    telem.Float64T,
		}
		if stamped {
			ch.LocalIndex = 1
		} else {
			ch.Virtual = true
		}
		channels[ch.Key()] = ch
		bindings = append(bindings, bind(f, ch.Key()))
		fmt.Fprintf(&doc, ",%q:1234.5678", name)
	}
	doc.WriteByte('}')
	m := jsonMessage("bench", fields...)
	rm := readOf(m, bindings...)
	if stamped {
		rm = indexed(ts, rm)
	}
	ic := icd.ICD{Name: "bench", Messages: []icd.Message{m}}
	s, err := read.NewSource([]icd.ReadMessage{rm}, ic, channels)
	if err != nil {
		b.Fatal(err)
	}
	return s, []byte(doc.String())
}

func BenchmarkSourceDecode(b *testing.B) {
	for _, stamped := range []bool{true, false} {
		for _, n := range []int{1, 10, 100} {
			layout := "virtual"
			if stamped {
				layout = "indexed"
			}
			b.Run(fmt.Sprintf("%s/fields=%d", layout, n), func(b *testing.B) {
				s, doc := benchSource(b, n, stamped)
				b.ReportAllocs()
				b.SetBytes(int64(len(doc)))
				for b.Loop() {
					if _, ok, err := s.Decode(doc); err != nil || !ok {
						b.Fatalf("decode: ok %t, err %v", ok, err)
					}
				}
			})
		}
	}
}
