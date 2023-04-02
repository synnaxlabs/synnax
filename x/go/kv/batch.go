// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv

import "context"

// Batch  is an ordered collection of key-value operations on the DB. Batch implements
// the Reader interface, and will read key-value pairs from both the Batch and underlying DB.
// A batch must be committed for its changes to be persisted.
type Batch interface {
	Writer
	Reader
	// Close closes the batch without committing it.
	Close() error
	// Commit persists the batch to the underlying DB.
	Commit(ctx context.Context, opts ...interface{}) error
}
