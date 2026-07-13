// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir

// BacksInternalChannel reports whether this kind is backed by a program-local
// channel (channel-read and literal) rather than a real external channel.
func (k VarKind) BacksInternalChannel() bool {
	return k == VarKindChannelRead || k == VarKindLiteral
}
