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

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp/migrate"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v2"
	projectv3 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
)

// MigrateLog lifts a v2 log to v3, clamping the timestamp precision to 0-9. It drops
// entries with channel 0 and keeps the first entry of each channel, as a later edit
// would otherwise fail validation.
func MigrateLog(ctx context.Context, old v2.Log) (Log, error) {
	out, err := autoMigrateLog(ctx, old)
	if err != nil {
		return Log{}, err
	}
	out.TimestampPrecision = uint8(min(max(old.TimestampPrecision, 0), 9))
	out.TimestampsHidden = old.ReceiptTimestampHidden
	out.TimeZone = telem.TimeZoneLocal
	out.Channels = lo.UniqBy(
		lo.Reject(
			out.Channels,
			func(e ChannelEntry, _ int) bool { return e.Channel == 0 },
		),
		func(e ChannelEntry) channel.Key { return e.Channel },
	)
	return out, nil
}

// MigrateChannelEntry lifts a v2 channel entry to v3. The v2 sentinel precision of -1
// becomes an absent precision, a larger one clamps to 17, an alias is cut to 64
// characters, and an empty display field takes its schema default.
func MigrateChannelEntry(
	ctx context.Context,
	old v2.ChannelEntry,
) (ChannelEntry, error) {
	out, err := autoMigrateChannelEntry(ctx, old)
	if err != nil {
		return ChannelEntry{}, err
	}
	if old.Precision >= 0 {
		precision := uint8(min(old.Precision, 17))
		out.Precision = &precision
	}
	out.Alias = cutToCharacters(old.Alias, 64)
	out.TimestampConfig = TimestampConfig{
		Format:   old.Timestamp.Format,
		TimeZone: old.Timestamp.Tz,
	}
	out.ApplyDefaults()
	return out, nil
}

func cutToCharacters(s string, n int) string {
	count := 0
	for i := range s {
		if count == n {
			return s[:i]
		}
		count++
	}
	return s
}

// Migration lifts stored logs from v2 to v3.
var Migration = migrate.NewEntry("v61_optional_channel_precision", MigrateLog)

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
		NameOf:   func(e Log) string { return e.Name },
		SetOwner: func(e *Log, p, r *uuid.UUID) { e.Project, e.Range = p, r },
	},
)
