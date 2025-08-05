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

// ErrAlwaysFailWriter is the error returned by AlwaysFailWriter.Write.
var ErrAlwaysFailWriter = errors.New("calls to AlwaysFailWriter.Write always fail")
var ErrAlwaysFailReader = errors.New("calls to AlwaysFailReader.Read always fail")

// AlwaysFailWriter is a writer that always fails. It is useful for testing the behavior
// of a writer when it is in error.
var AlwaysFailWriter = &alwaysFailWriter{}

type alwaysFailWriter struct{}

var _ io.Writer = (*alwaysFailWriter)(nil)

func (w *alwaysFailWriter) Write([]byte) (int, error) { return 0, ErrAlwaysFailWriter }

var AlwaysFailReader = &alwaysFailReader{}

type alwaysFailReader struct{}

var _ io.Reader = (*alwaysFailReader)(nil)

func (r *alwaysFailReader) Read([]byte) (int, error) { return 0, ErrAlwaysFailReader }
