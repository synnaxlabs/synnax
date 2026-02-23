// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

import "context"

// Codec defines a custom encoding/decoding strategy for entries stored in a Table.
// When a Codec is set on a Table, it takes precedence over the default DB codec
// for value encoding/decoding.
type Codec[E any] interface {
	Marshal(ctx context.Context, entry E) ([]byte, error)
	Unmarshal(ctx context.Context, data []byte) (E, error)
}
