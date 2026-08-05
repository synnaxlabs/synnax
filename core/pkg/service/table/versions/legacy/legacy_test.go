// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package legacy_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/table/versions/legacy"
	v0 "github.com/synnaxlabs/synnax/pkg/service/table/versions/legacy/v0"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

func jsonMap(raw string) msgpack.EncodedJSON {
	var m map[string]any
	Expect(json.Unmarshal([]byte(raw), &m)).To(Succeed())
	return m
}

var _ = Describe("MigrateData", func() {
	It("Should decode an explicit 0.0.0 blob into v0.Data", func() {
		out := MustSucceed(legacy.MigrateData(jsonMap(`{
			"version": "0.0.0",
			"layout": {
				"rows": [{"size": 36, "cells": [{"key": "a"}]}],
				"columns": [{"size": 72}]
			},
			"cells": {"a": {"key": "a", "variant": "text", "props": {"value": "hi"}}}
		}`)))
		Expect(out.Layout.Rows).To(HaveLen(1))
		Expect(out.Layout.Rows[0].Cells[0].Key).To(Equal("a"))
		Expect(out.Cells["a"].Variant).To(Equal("text"))
	})

	It("Should fall back to v0 when the blob has no version field", func() {
		out := MustSucceed(legacy.MigrateData(jsonMap(`{
			"layout": {"rows": [], "columns": [{"size": 80}]},
			"cells": {}
		}`)))
		Expect(out.Layout.Columns).To(HaveLen(1))
		Expect(out.Layout.Columns[0].Size).To(Equal(80.0))
	})

	It("Should decode a nil blob into a zero v0.Data", func() {
		out := MustSucceed(legacy.MigrateData(nil))
		Expect(out).To(Equal(v0.Data{}))
	})

	It("Should error on an unknown declared version", func() {
		Expect(legacy.MigrateData(jsonMap(`{"version": "99.0.0"}`))).Error().
			To(MatchError(ContainSubstring("unknown table data version")))
	})
})
