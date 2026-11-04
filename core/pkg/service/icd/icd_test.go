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

// checkImport fails t unless err is a validation error at the data path, or the
// messages have unique keys, references that resolve, and finite numbers.
func checkImport(t *testing.T, messages []icd.Message, err error) {
	t.Helper()
	if err != nil {
		var pathErr validate.PathError
		if !errors.As(err, &pathErr) || !slices.Equal(pathErr.Path, []string{"data"}) ||
			!errors.Is(pathErr.Err, validate.ErrValidation) {
			t.Fatalf("not a validation error at the data path: %v", err)
		}
		return
	}
	keys := make(set.Set[icd.MessageKey], len(messages))
	for _, m := range messages {
		if keys.Contains(m.Key) {
			t.Fatalf("duplicate message key %s", m.Key)
		}
		keys.Add(m.Key)
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
			if f.Multiplexor != nil && !fields.Contains(*f.Multiplexor) {
				t.Fatalf("field %s.%s has no multiplexor field", m.Name, f.Name)
			}
		}
	}
}

var _ = Describe("Parse", func() {
	DescribeTable("Should parse each format",
		func(format icd.Format, data func() []byte, count int) {
			Expect(MustSucceed(icd.Parse(format, data()))).To(HaveLen(count))
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
