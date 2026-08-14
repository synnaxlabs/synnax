// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package control publishes the cluster's control state on a channel and answers
// control state reads. It is the only writer of the sy_control channel.
package control

import "github.com/synnaxlabs/synnax/pkg/distribution/control"

type (
	// Subject is an entity that can hold control over a channel.
	Subject = control.Subject
	// State is the control state of a single channel: the subject holding authority
	// over it, and the level of that authority.
	State = control.State
	// Transfer is a transition of control over a single channel. A nil From is an
	// acquire; a nil To is a release.
	Transfer = control.Transfer
	// Update is a batch of transfers that occurred atomically.
	Update = control.Update
)

// ChannelName is the name of the free virtual channel carrying control updates for the
// entire cluster.
const ChannelName = "sy_control"
