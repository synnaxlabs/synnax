// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package domain_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Errors", func() {
	Describe("NewErrRangeWriteConflict", func() {
		It("Should correctly format a range overlap error", func() {
			writerTr := (telem.SecondTS * 5).SpanRange(3 * telem.Second)
			existingTr := (telem.SecondTS * 2).SpanRange(4 * telem.Second)
			err := domain.NewErrRangeWriteConflict(writerTr, existingTr)
			Expect(err).To(MatchError(domain.ErrWriteConflict))
			Expect(err.Error()).To(Equal("write for range 1970-01-01T00:00:05Z - :08 (3s) overlaps with existing data occupying time range 1970-01-01T00:00:02Z - :06 (4s) for a time span of 1s: write overlaps with existing data in database: validation error"))
		})
	})
	Describe("NewErrPointWriteConflict", func() {
		It("Should correctly format a point overlap error", func() {
			existingTr := (telem.SecondTS * 2).SpanRange(4 * telem.Second)
			point := (2 * telem.SecondTS).Add(1 * telem.Second)
			err := domain.NewErrPointWriteConflict(point, existingTr)
			Expect(err).To(MatchError(domain.ErrWriteConflict))
			Expect(err.Error()).To(Equal("1970-01-01T00:00:03Z overlaps with existing data occupying time range 1970-01-01T00:00:02Z - :06 (4s). Timestamp occurs 1s after the start and 3s before the end of the range: write overlaps with existing data in database: validation error"))
		})
	})
})
