// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/codec"
	"github.com/synnaxlabs/synnax/pkg/service/icd/read"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// dataTypeTable is testdata/data_types.json, which the Console's data type spec reads
// too. Reads and defaults hold the cases of each format, and writes are of JSON fields.
type dataTypeTable struct {
	Payloads map[string]map[string]any `json:"payloads"`
	Fields   map[string]map[string]any `json:"fields"`
	Reads    map[string][]dataTypeCase `json:"reads"`
	Writes   []dataTypeCase            `json:"writes"`
	Defaults map[string][]dataTypeCase `json:"defaults"`
}

// dataTypeCase is a field, a data type, and whether the Core accepts them. A default
// omits accepted, as the Core must accept it.
type dataTypeCase struct {
	Field    map[string]any `json:"field"`
	DataType telem.DataType `json:"data_type"`
	Accepted bool           `json:"accepted"`
}

// payloadOf returns the payload of format in t that holds the field of c, and the key
// of the field.
func payloadOf(
	t dataTypeTable,
	format string,
	c dataTypeCase,
) (icd.Payload, icd.FieldKey) {
	field := maps.Clone(t.Fields[format])
	maps.Copy(field, c.Field)
	raw := maps.Clone(t.Payloads[format])
	raw["fields"] = []any{field}
	var (
		p    icd.Payload
		base icd.BaseField
	)
	Expect(json.Unmarshal(MustSucceed(json.Marshal(raw)), &p)).To(Succeed())
	Expect(json.Unmarshal(MustSucceed(json.Marshal(field)), &base)).To(Succeed())
	return p, base.Key
}

func caseName(format string, c dataTypeCase) string {
	return fmt.Sprintf("a %s field %v and %s", format, c.Field, c.DataType)
}

// checkRead returns the error of a read of the field of c into a channel of its data
// type.
func checkRead(
	ctx context.Context,
	t dataTypeTable,
	format string,
	c dataTypeCase,
) error {
	p, key := payloadOf(t, format, c)
	msg := icd.Message{Key: uuid.New(), Name: "message", Payload: p}
	ic := icd.ICD{Key: uuid.New(), Name: "ICD", Messages: []icd.Message{msg}}
	resolver := read.NewResolver(svc, func(
		context.Context,
		gorp.Snapshot,
		[]channel.Key,
	) (map[channel.Key]icdv0.Channel, error) {
		return nil, nil
	})
	return resolver.Check(ctx, db, []read.Read{{
		Path: []string{"reads", "0"},
		ICD:  ic.Key,
		// Only a framed read takes a message that is not JSON.
		Framed: format != "json",
		Messages: []icd.ReadMessage{{
			Message:  msg.Key,
			Bindings: []icd.ReadBinding{{Field: key, DataType: c.DataType}},
		}},
	}}, map[icd.Key]icd.ICD{ic.Key: ic})
}

// checkWrite returns the error of an encoder of the JSON field of c from a channel of
// its data type.
func checkWrite(t dataTypeTable, c dataTypeCase) error {
	p, key := payloadOf(t, "json", c)
	jp, ok := p.Variant.(icd.JSONPayload)
	Expect(ok).To(BeTrue())
	_, err := codec.NewJSONEncoder(codec.JSONMessage{
		Name:     "message",
		Payload:  jp,
		Bindings: []codec.Binding{{Field: key, DataType: c.DataType}},
	})
	return err
}

// entries returns an entry for each case of byFormat, in an order that every parallel
// process builds alike.
func entries(byFormat map[string][]dataTypeCase) []TableEntry {
	var out []TableEntry
	for _, format := range slices.Sorted(maps.Keys(byFormat)) {
		for _, c := range byFormat[format] {
			out = append(out, Entry(caseName(format, c), format, c))
		}
	}
	return out
}

var _ = Describe("Data types", func() {
	var t dataTypeTable
	Expect(json.Unmarshal(
		MustSucceed(os.ReadFile("testdata/data_types.json")),
		&t,
	)).To(Succeed())
	reads, defaults := entries(t.Reads), entries(t.Defaults)
	writes := make([]TableEntry, len(t.Writes))
	for i, c := range t.Writes {
		writes[i] = Entry(caseName("json", c), c)
	}

	DescribeTable("Should read a field into the data types of the table",
		func(ctx SpecContext, format string, c dataTypeCase) {
			err := checkRead(ctx, t, format, c)
			if c.Accepted {
				Expect(err).To(Succeed())
				return
			}
			Expect(err).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(
					ContainSubstring("cannot read into data type %s", c.DataType),
				),
			))
		},
		reads,
	)

	DescribeTable("Should write a JSON field from the data types of the table",
		func(c dataTypeCase) {
			err := checkWrite(t, c)
			if c.Accepted {
				Expect(err).To(Succeed())
				return
			}
			Expect(err).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(Or(
					ContainSubstring("cannot write %s channels", c.DataType),
					ContainSubstring("requires a float channel, not %s", c.DataType),
				)),
			))
		},
		writes,
	)

	DescribeTable("Should accept the default data type of a field",
		func(ctx SpecContext, format string, c dataTypeCase) {
			Expect(checkRead(ctx, t, format, c)).To(Succeed())
			if format == "json" {
				Expect(checkWrite(t, c)).To(Succeed())
			}
		},
		defaults,
	)
})
