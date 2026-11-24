// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"fmt"
	"math"
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
)

type (
	breaker3Int   int64
	breaker3Uint  uint32
	breaker3Float float64
)

var _ = Describe("Channel reference breaker 3", func() {
	// NaN, the infinities, and complex values are not in the corpus: the JSON encoding
	// of an Arc refuses them, so no node input holds them.
	It(
		"Should reference the channel that the graph compiler's coerce reads from a "+
			"node input",
		func(ctx SpecContext) {
			ch := channel.Channel{
				Name:     "rb3_channel",
				Virtual:  true,
				DataType: telem.Float64T,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			k := ch.Key()
			one := 1
			f := 42.0
			corpus := []any{
				0,
				1,
				-1,
				42,
				math.MaxUint32,
				math.MaxUint32 + 1,
				math.MinInt64,
				int8(-1),
				int8(7),
				int16(300),
				int32(-5),
				int32(k),
				int64(0),
				int64(k),
				int64(math.MaxUint32),
				int64(math.MaxUint32 + 1),
				int64(math.MaxInt64),
				int64(math.MinInt64),
				uint(0),
				uint(9),
				uint8(255),
				uint16(65535),
				uint32(0),
				uint32(1),
				uint32(math.MaxUint32),
				uint64(math.MaxUint32),
				uint64(math.MaxUint32 + 1),
				uint64(math.MaxUint64),
				uint64(1 << 63),
				float32(0),
				float32(math.Copysign(0, -1)),
				float32(1),
				float32(1.5),
				float32(4294967040),
				float32(4294967296),
				float32(16777217),
				float32(math.SmallestNonzeroFloat32),
				float32(-1),
				0.0,
				math.Copysign(0, -1),
				1.0,
				float64(k),
				float64(k) + 0.5,
				float64(k) + 0.0000001,
				float64(math.MaxUint32),
				math.MaxUint32 + 0.5,
				math.MaxUint32 + 1.0,
				math.Nextafter(math.MaxUint32, math.Inf(1)),
				1e300,
				-1e300,
				-1.0,
				math.SmallestNonzeroFloat64,
				math.MaxFloat64,
				float64(1 << 53),
				float64(1<<53 + 2),
				breaker3Int(42),
				breaker3Int(-42),
				breaker3Uint(42),
				breaker3Float(42),
				breaker3Float(42.5),
				true,
				false,
				nil,
				&one,
				&f,
				[]any{42},
				map[string]any{"key": 42},
				struct{}{},
			}
			// Uses takes keys, so the keys that coerce reads and the keys that a wrong
			// read of the corpus could give stand for every channel.
			checked := []channel.Key{1, 42, k}
			type read struct {
				key       channel.Key
				ok        bool
				arc       string
				createErr error
			}
			reads := make([]read, len(corpus))
			for i, v := range corpus {
				var n uint32
				zerr := zyn.Uint32().Coerce().Parse(v, &n)
				r := read{key: channel.Key(n), ok: zerr == nil && n != 0}
				if r.ok {
					checked = append(checked, r.key)
				}
				a := newAutomation(arc.Base{
					Name: fmt.Sprintf("rb3_%d", i),
					Source: arc.Source{Variant: arc.GraphSource{Graph: graph.Graph{
						Nodes: []graph.Node{{Key: "on"}},
						Inputs: map[string]msgpack.EncodedJSON{
							"on": {"type": "on", "channel": v},
						},
					}}},
				})
				r.createErr = db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &a)
				})
				if r.createErr == nil {
					key := a.Base().Key
					r.arc = key.String()
					DeferCleanup(func(ctx SpecContext) {
						Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
							return svc.NewWriter(tx).Delete(ctx, key)
						})).To(Succeed())
					})
				}
				reads[i] = r
			}
			slices.Sort(checked)
			checked = slices.Compact(checked)
			uses := MustSucceed(channelSvc.Uses(ctx, db, checked))
			held := make(map[string][]channel.Key)
			for i, used := range uses {
				for _, u := range used {
					held[u.Holder.Key] = append(held[u.Holder.Key], checked[i])
				}
			}
			var mismatches []string
			for i, r := range reads {
				v := corpus[i]
				switch {
				case r.createErr != nil:
					never := fmt.Sprintf("channel %d was never issued", r.key)
					if !r.ok || !errors.Is(r.createErr, validate.ErrValidation) ||
						!strings.Contains(r.createErr.Error(), never) {
						mismatches = append(mismatches, fmt.Sprintf(
							"%T(%v): coerce (%d, %v), create %v",
							v, v, r.key, r.ok, r.createErr,
						))
					}
				case r.ok && !slices.Equal(held[r.arc], []channel.Key{r.key}),
					!r.ok && len(held[r.arc]) > 0:
					mismatches = append(mismatches, fmt.Sprintf(
						"%T(%v): coerce (%d, %v), Arc holds %v",
						v, v, r.key, r.ok, held[r.arc],
					))
				}
			}
			Expect(mismatches).To(BeEmpty())
		},
	)
})
