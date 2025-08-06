// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package io

import (
	"io"

	"github.com/synnaxlabs/x/errors"
)

var (
	// ErrFailWriter is the error returned by FailWriter.Write.
	ErrFailWriter = errors.New("calls to FailWriter.Write always fail")
	// ErrFailReader is the error returned by FailReader.Read.
	ErrFailReader = errors.New("calls to FailReader.Read always fail")
)

var (
	// FailWriter is a writer that always fails. It is useful for testing the behavior
	// of a writer when it is in error.
	FailWriter = &failWriter{}
	// FailReader is a reader that always fails. It is useful for testing the behavior
	// of a reader when it is in error.
	FailReader = &failReader{}
)

type failWriter struct{}

var _ io.Writer = (*failWriter)(nil)

func (w *failWriter) Write([]byte) (int, error) { return 0, ErrFailWriter }

type failReader struct{}

var _ io.Reader = (*failReader)(nil)

func (r *failReader) Read([]byte) (int, error) { return 0, ErrFailReader }
