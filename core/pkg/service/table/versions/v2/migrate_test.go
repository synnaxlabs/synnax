// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	"encoding/json"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v2"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	text "github.com/synnaxlabs/x/text/versions/v0"
)

// jsonMap parses raw as the opaque props record a v1 cell stores.
func jsonMap(raw string) msgpack.EncodedJSON {
	GinkgoHelper()
	var m map[string]any
	Expect(json.Unmarshal([]byte(raw), &m)).To(Succeed())
	return m
}

// cfgFields returns a typed cell config's wire fields for shape assertions.
func cfgFields(cfg v2.CellConfig) map[string]any {
	GinkgoHelper()
	var m map[string]any
	Expect(json.Unmarshal(MustSucceed(json.Marshal(cfg)), &m)).To(Succeed())
	return m
}

var _ = Describe("MigrateTable", func() {
	// migrateCell lifts a single v1 cell and returns its typed config.
	migrateCell := func(ctx SpecContext, variant, props string) v2.CellConfig {
		GinkgoHelper()
		out := MustSucceed(v2.MigrateTable(ctx, v1.Table{
			Key:  uuid.New(),
			Name: "t",
			Cells: map[string]v1.Cell{
				"a": {Key: "a", Variant: variant, Props: jsonMap(props)},
			},
		}))
		Expect(out.Cells).To(HaveKey("a"))
		return out.Cells["a"]
	}

	It("Should pass the structural fields through untouched", func(ctx SpecContext) {
		key := uuid.New()
		out := MustSucceed(v2.MigrateTable(ctx, v1.Table{
			Key:     key,
			Name:    "trip-table",
			Rows:    []v1.Row{{Size: 36, Cells: []string{"a", "b"}}},
			Columns: []v1.Column{{Size: 72}, {Size: 100}},
		}))
		Expect(out.Key).To(Equal(key))
		Expect(out.Name).To(Equal("trip-table"))
		Expect(out.Rows).To(Equal([]v2.Row{{Size: 36, Cells: []string{"a", "b"}}}))
		Expect(out.Columns).To(Equal([]v2.Column{{Size: 72}, {Size: 100}}))
	})

	It("Should produce an empty cells map for a table with none", func(
		ctx SpecContext,
	) {
		out := MustSucceed(v2.MigrateTable(ctx, v1.Table{Key: uuid.New()}))
		Expect(out.Cells).NotTo(BeNil())
		Expect(out.Cells).To(BeEmpty())
	})

	It("Should normalize camelCase props to the snake_case wire form", func(
		ctx SpecContext,
	) {
		fields := cfgFields(migrateCell(
			ctx, "text", `{"value": "hi", "backgroundColor": "#112233"}`,
		))
		Expect(fields).To(HaveKeyWithValue("value", "hi"))
		Expect(fields).To(HaveKey("background_color"))
		Expect(fields).NotTo(HaveKey("backgroundColor"))
	})

	It("Should fill absent fields with their schema defaults", func(ctx SpecContext) {
		cfg, ok := migrateCell(ctx, "text", `{}`).Variant.(v2.TextCellConfig)
		Expect(MustBeOk(cfg, ok).Level).To(Equal(text.LevelH5))
		Expect(cfg.Weight).To(Equal(400.0))
		Expect(cfg.Align).To(Equal(v2.FlexAlignmentCenter))
	})

	It("Should extract value cell args from the stored pipeline spec", func(
		ctx SpecContext,
	) {
		fields := cfgFields(migrateCell(ctx, "value", `{
			"telem": {
				"type": "source-pipeline",
				"variant": "source",
				"valueType": "string",
				"props": {
					"segments": {
						"valueStream": {"props": {"channel": 42}},
						"rollingAverage": {"props": {"windowSize": 5}},
						"stringifier": {"props": {"precision": 3, "notation": "scientific"}}
					},
					"outlet": "stringifier"
				}
			},
			"units": "psi"
		}`))
		Expect(fields).To(SatisfyAll(
			HaveKeyWithValue("variant", "value"),
			HaveKeyWithValue("channel", 42.0),
			HaveKeyWithValue("rolling_average", 5.0),
			HaveKeyWithValue("precision", 3.0),
			HaveKeyWithValue("notation", "scientific"),
			HaveKeyWithValue("units", "psi"),
		))
		Expect(fields).NotTo(HaveKey("telem"))
	})

	It("Should leave the channel at the zero sentinel for a legacy zero channel", func(
		ctx SpecContext,
	) {
		fields := cfgFields(migrateCell(ctx, "value", `{
			"telem": {"props": {"segments": {"valueStream": {"props": {"channel": 0}}}}}
		}`))
		Expect(fields).To(HaveKeyWithValue("channel", 0.0))
	})

	DescribeTable("Should map legacy x-location alignments onto flex alignments",
		func(ctx SpecContext, legacy, want string) {
			fields := cfgFields(migrateCell(
				ctx, "text", `{"value": "hi", "align": "`+legacy+`"}`,
			))
			Expect(fields).To(HaveKeyWithValue("align", want))
		},
		Entry("left", "left", "start"),
		Entry("right", "right", "end"),
		Entry("center", "center", "center"),
		Entry("start passes through", "start", "start"),
		Entry("end passes through", "end", "end"),
	)

	It("Should map named font weights onto numeric weights", func(ctx SpecContext) {
		fields := cfgFields(migrateCell(ctx, "text", `{"weight": "bold"}`))
		Expect(fields).To(HaveKeyWithValue("weight", 700.0))
	})

	It("Should degrade a cell with an unknown variant to an empty text cell", func(
		ctx SpecContext,
	) {
		fields := cfgFields(migrateCell(ctx, "hologram", `{"value": "hi"}`))
		Expect(fields).To(HaveKeyWithValue("variant", "text"))
		Expect(fields).To(HaveKeyWithValue("value", ""))
	})
})
