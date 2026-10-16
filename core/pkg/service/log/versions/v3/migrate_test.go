// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v2 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/notation"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func newV2() v2.Log {
	return v2.Log{
		Key:  uuid.New(),
		Name: "my-log",
		Channels: []v2.ChannelEntry{{
			Channel:   42,
			Color:     color.MustFromHex("#ff0000"),
			Notation:  notation.NotationScientific,
			Precision: 3,
			Alias:     "temp",
			Timestamp: v2.TimestampConfig{
				Format: telem.TimestampFormatTime,
				Tz:     telem.TimeZoneUTC,
			},
		}},
		TimestampPrecision:     2,
		ChannelNamesHidden:     true,
		ReceiptTimestampHidden: true,
	}
}

var _ = Describe("MigrateLog", func() {
	It("Should carry every v2 field and default the range", func(ctx SpecContext) {
		in := newV2()
		out := MustSucceed(v3.MigrateLog(ctx, in))
		Expect(out.Key).To(Equal(in.Key))
		Expect(out.Name).To(Equal(in.Name))
		Expect(out.Channels).To(Equal([]v3.ChannelEntry{{
			Channel:   42,
			Color:     color.MustFromHex("#ff0000"),
			Notation:  notation.NotationScientific,
			Precision: 3,
			Alias:     "temp",
			Timestamp: v3.TimestampConfig{
				Format: telem.TimestampFormatTime,
				Tz:     telem.TimeZoneUTC,
			},
		}}))
		Expect(out.TimestampPrecision).To(Equal(in.TimestampPrecision))
		Expect(out.ChannelNamesHidden).To(BeTrue())
		Expect(out.ReceiptTimestampHidden).To(BeTrue())
		Expect(out.Range).To(Equal(ranger.Ref{
			Variant: ranger.WindowRef{Window: telem.Window{
				Variant: telem.DynamicWindow{TimeSpan: 7 * telem.Day},
			}},
		}))
	})

	It("Should round-trip the migrated log through the v3 codec",
		func(ctx SpecContext) {
			out := MustSucceed(v3.MigrateLog(ctx, newV2()))
			encoded := MustSucceed(orc.Codec.Encode(ctx, out))
			var decoded v3.Log
			Expect(orc.Codec.Decode(ctx, encoded, &decoded)).To(Succeed())
			Expect(decoded).To(Equal(out))
		})
})

var _ = Describe("Storage migration", func() {
	It("Should lift a stored v2 entry to v3 through the gorp migration",
		func(ctx SpecContext) {
			seed := newV2()
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v2.Key, v2.Log]{DB: db}))
			Expect(gorp.NewCreate[v2.Key, v2.Log]().
				Entry(&seed).Exec(ctx, db)).To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "Log",
				Migrations: []migrate.Migration{v3.Migration},
			})).To(Succeed())
			var got v3.Log
			Expect(gorp.NewRetrieve[v3.Key, v3.Log]().
				Where(gorp.MatchKeys[v3.Key, v3.Log](seed.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got).To(Equal(MustSucceed(v3.MigrateLog(ctx, seed))))
		})
})
