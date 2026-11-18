// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package payload_test

import (
	"encoding/json/jsontext"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
)

var _ = DescribeTable(
	"Absent",
	func(raw jsontext.Value, dt telem.DataType, absent bool) {
		Expect(payload.Absent(raw, dt)).To(Equal(absent))
	},
	Entry("no value", jsontext.Value(nil), telem.Float64T, true),
	Entry("null", jsontext.Value("null"), telem.StringT, true),
	Entry("an empty string on a number", jsontext.Value(`""`), telem.Int8T, true),
	Entry(
		"an empty string on a string",
		jsontext.Value(`""`),
		telem.StringT,
		false,
	),
	Entry("zero", jsontext.Value("0"), telem.Float64T, false),
	Entry("null on JSON", jsontext.Value("null"), telem.JSONT, false),
	Entry("an empty string on JSON", jsontext.Value(`""`), telem.JSONT, false),
	Entry("an empty string on bytes", jsontext.Value(`""`), telem.BytesT, false),
	Entry("null on bytes", jsontext.Value("null"), telem.BytesT, true),
	Entry("an empty string on a UUID", jsontext.Value(`""`), telem.UUIDT, true),
)
