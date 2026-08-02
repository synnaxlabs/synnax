// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

// Chan returns a channel type wrapping the given value type.
func Chan(valueType Type) Type {
	return Type{Kind: KindChan, Elem: &valueType}
}

// ReadChan returns a channel type annotated for read access.
func ReadChan(valueType Type) Type {
	return Type{Kind: KindChan, Elem: &valueType, ChanDirection: ChanDirectionRead}
}

// WriteChan returns a channel type annotated for write access.
func WriteChan(valueType Type) Type {
	return Type{Kind: KindChan, Elem: &valueType, ChanDirection: ChanDirectionWrite}
}

// NewChannels creates a new Channels with empty read and write sets.
func NewChannels() Channels {
	return Channels{Read: make(map[uint32]string), Write: make(map[uint32]string)}
}
