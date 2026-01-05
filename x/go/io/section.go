// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package io

import "io"

type SectionReaderAtCloser struct {
	io.ReaderAt
	io.Closer
}

func NewSectionReaderAtCloser(r ReaderAtCloser, off int64, n int64) *SectionReaderAtCloser {
	sectionReader := io.NewSectionReader(r, off, n)
	return &SectionReaderAtCloser{ReaderAt: sectionReader, Closer: r}
}
