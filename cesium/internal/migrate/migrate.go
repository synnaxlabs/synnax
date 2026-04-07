// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrate

import (
	"fmt"

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/version"
	"github.com/synnaxlabs/x/io/fs"
)

// DBState is metadata about a single-channel database that can be migrated. This data
// structure is passed into migration functions on bootup.
type DBState struct {
	// FS is the file-system for that channel in the DB. This is not the
	// top level cesium directory, but the channel-specific directory itself.
	FS fs.FS
	// Channel is the channel specification for the DB.
	Channel channel.Channel
	// ShouldIgnoreChannel can be set to true by the migration function if the channel
	// should be ignored on database startup.
	ShouldIgnoreChannel bool
}

type migration func(state DBState) DBState

var migrations = []migration{
	migrateV0toV1,
	migrateV1toV2,
}

func migrateV0toV1(state DBState) DBState {
	state.Channel.Version = version.Version1
	if state.Channel.Name == "" {
		state.Channel.Name = fmt.Sprintf("Unknown %v", state.Channel.Key)
	}
	return state
}

func migrateV1toV2(state DBState) DBState {
	state.Channel.Version = version.Version2
	if state.Channel.Virtual || state.Channel.IsIndex {
		return state
	}
	state.ShouldIgnoreChannel = state.Channel.Index == 0
	return state
}

// Migrate runs all pending version migrations on the given DBState, starting from the
// channel's current version up to the latest.
func Migrate(state DBState) DBState {
	for i := int(state.Channel.Version); i < len(migrations); i++ {
		state = migrations[i](state)
	}
	return state
}
