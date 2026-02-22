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

// GorpMarshaler is implemented by entry types that need custom serialization
// when stored via gorp. When an entry implements this interface, gorp's Writer
// will call GorpMarshal instead of the codec's Encode method.
type GorpMarshaler interface {
	GorpMarshal(ctx context.Context) ([]byte, error)
}

// GorpUnmarshaler is implemented by entry types that need custom deserialization
// when read via gorp. When an entry implements this interface, gorp's Reader
// and Iterator will call GorpUnmarshal instead of the codec's Decode method.
type GorpUnmarshaler interface {
	GorpUnmarshal(ctx context.Context, data []byte) error
}
