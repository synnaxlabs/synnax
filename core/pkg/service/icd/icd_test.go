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
	"math"
	"slices"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// checkImport fails t unless err is a validation error at the data path, or the entries
// have unique keys, references that resolve, finite numbers, and positive periods.
func checkImport(t *testing.T, entries []icd.Entry, err error) {
	t.Helper()
	if err != nil {
		var pathErr validate.PathError
		if !errors.As(err, &pathErr) || !slices.Equal(pathErr.Path, []string{"data"}) ||
			!errors.Is(pathErr.Err, validate.ErrValidation) {
			t.Fatalf("not a validation error at the data path: %v", err)
		}
		return
	}
	keys := make(set.Set[icd.EntryKey], len(entries))
	enums := make(set.Set[icd.EntryKey])
	var messages []icd.MessageEntry
	for _, e := range entries {
		var key icd.EntryKey
		switch v := e.Variant.(type) {
		case icd.EnumEntry:
			key = v.Key
			enums.Add(key)
		case icd.MessageEntry:
			key = v.Key
			messages = append(messages, v)
		default:
			t.Fatalf("unexpected entry %T", v)
		}
		if keys.Contains(key) {
			t.Fatalf("duplicate entry key %s", key)
		}
		keys.Add(key)
	}
	for _, m := range messages {
		if m.Period != nil && *m.Period <= 0 {
			t.Fatalf("message %s has period %s", m.Name, *m.Period)
		}
		p, ok := m.Payload.Variant.(icd.BinaryPayload)
		if !ok {
			t.Fatalf("message %s has payload %T", m.Name, m.Payload.Variant)
		}
		fields := make(set.Set[icd.FieldKey], len(p.Fields))
		for _, f := range p.Fields {
			if fields.Contains(f.Key) {
				t.Fatalf("message %s has duplicate field key %s", m.Name, f.Key)
			}
			fields.Add(f.Key)
		}
		for _, f := range p.Fields {
			for _, v := range []*float64{&f.Scale, &f.Offset, f.Min, f.Max} {
				if v != nil && (math.IsNaN(*v) || math.IsInf(*v, 0)) {
					t.Fatalf(
						"field %s.%s has a number that is not finite",
						m.Name,
						f.Name,
					)
				}
			}
			if f.Enumeration != nil && !enums.Contains(*f.Enumeration) {
				t.Fatalf("field %s.%s has no enum entry", m.Name, f.Name)
			}
			if f.Multiplexor != nil && !fields.Contains(*f.Multiplexor) {
				t.Fatalf("field %s.%s has no multiplexor field", m.Name, f.Name)
			}
		}
	}
}

var _ = Describe("Parse", func() {
	DescribeTable("Should parse each format",
		func(format icd.Format, data func() []byte, messages int) {
			entries := MustSucceed(icd.Parse(format, data()))
			Expect(messagesOf(entries)).To(HaveLen(messages))
		},
		Entry("DBC", icd.FormatDBC, func() []byte {
			data, _ := readFixture("basic")
			return data
		}, 5),
		Entry("CSV", icd.FormatCSV, func() []byte { return toCSV(tableRows) }, 3),
		Entry("XLSX", icd.FormatXLSX, func() []byte { return toXLSX(tableRows) }, 3),
	)

	It("Should reject an unknown format", func() {
		Expect(icd.Parse("arxml", nil)).Error().To(MatchError(
			`format: unsupported format "arxml": validation error`,
		))
	})
})
