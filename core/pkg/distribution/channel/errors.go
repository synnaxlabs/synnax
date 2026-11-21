// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import "fmt"

// Error is an error of one channel. It carries the key of the channel, never its name,
// so a caller that holds channel records can name it.
type Error struct {
	Err error
	Key Key
}

// Error implements error.
func (e Error) Error() string { return fmt.Sprintf("channel <%d>: %v", e.Key, e.Err) }

// Unwrap returns the error of the channel.
func (e Error) Unwrap() error { return e.Err }
