// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp/migrate"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v2"
	projectv3 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// MigrateLog lifts a v2 log to v3.
func MigrateLog(ctx context.Context, old v2.Log) (Log, error) {
	out, err := autoMigrateLog(ctx, old)
	if err != nil {
		return Log{}, err
	}
	out.TimestampsHidden = old.ReceiptTimestampHidden
	return out, nil
}

// MigrateChannelEntry lifts a v2 channel entry to v3. The v2 sentinel precision of -1
// becomes an absent precision.
func MigrateChannelEntry(
	ctx context.Context,
	old v2.ChannelEntry,
) (ChannelEntry, error) {
	out, err := autoMigrateChannelEntry(ctx, old)
	if err != nil {
		return ChannelEntry{}, err
	}
	if old.Precision >= 0 {
		precision := old.Precision
		out.Precision = &precision
	}
	return out, nil
}

// Migration lifts stored logs from v2 to v3.
var Migration = migrate.NewEntry("v60_optional_channel_precision", MigrateLog)

// MembersMigration sets the group of each stored log from the ontology edge that placed
// it in a group.
var MembersMigration = groupv1.NewMembersMigration(groupv1.MembersConfig[Key, Log]{
	Name:         "v61_log_group_membership",
	Type:         resource.TypeLog,
	SetGroup:     func(e *Log, g groupv1.Key) { e.Group = &g },
	ProjectOwned: true,
})

// OwnersMigration sets the project or range of each stored log from the ontology edge
// that owned it, and deletes each log with no owner.
var OwnersMigration = projectv3.NewOwnersMigration(
	projectv3.OwnersConfig[Key, Log]{
		Name:     "v61_log_owner",
		Type:     resource.TypeLog,
		Owned:    func(e Log) bool { return e.Project != nil || e.Range != nil },
		SetOwner: func(e *Log, p, r *uuid.UUID) { e.Project, e.Range = p, r },
	},
)
