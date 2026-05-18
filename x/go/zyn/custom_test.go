// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package zyn_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/zyn"
)

type hex int

var _ = Describe("Custom", func() {
	parseHex := func(data any) (hex, error) {
		s, ok := data.(string)
		if !ok {
			return 0, errors.Newf("expected string, got %T", data)
		}
		n, err := strconv.ParseInt(s, 16, 32)
		if err != nil {
			return 0, err
		}
		return hex(n), nil
	}

	It("Should parse wire data into the typed destination", func() {
		schema := zyn.Custom(parseHex)
		var out hex
		MustSucceed(struct{}{}, schema.Parse("ff", &out))
		Expect(out).To(Equal(hex(255)))
	})

	It("Should surface the parser's error verbatim", func() {
		schema := zyn.Custom(parseHex)
		var out hex
		Expect(schema.Parse(42, &out)).To(MatchError(ContainSubstring("expected string")))
	})

	It("Should leave the destination at zero when optional and data is nil", func() {
		schema := zyn.Custom(parseHex).Optional()
		out := hex(99)
		Expect(schema.Parse(nil, &out)).To(Succeed())
	})

	It("Should reject nil when not optional", func() {
		schema := zyn.Custom(parseHex)
		var out hex
		Expect(schema.Parse(nil, &out)).Error().To(HaveOccurred())
	})

	It("Should dump as identity when no dump function is configured", func() {
		schema := zyn.Custom(parseHex)
		Expect(schema.Dump(hex(255))).To(Equal(hex(255)))
	})

	It("Should dump via the configured function when present", func() {
		schema := zyn.Custom(parseHex).WithDump(func(h hex) (any, error) {
			return strconv.FormatInt(int64(h), 16), nil
		})
		Expect(schema.Dump(hex(255))).To(Equal("ff"))
	})
})
