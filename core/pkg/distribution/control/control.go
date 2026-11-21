// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package control makes the control state of channels readable and observable. The
// storage engine arbitrates control.
package control

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/control"
)

type (
	// Subject is an entity that can hold control authority over a channel.
	Subject = control.Subject
	// State is the control state of a single channel: the subject holding authority
	// over it, and the level of that authority.
	State = control.State[channel.Key]
	// Transfer is a transition of control over a single channel. A nil From is an
	// acquire; a nil To is a release.
	Transfer = control.Transfer[channel.Key]
	// Update is a batch of transfers that a Core applied atomically.
	Update = control.Update[channel.Key]
)
