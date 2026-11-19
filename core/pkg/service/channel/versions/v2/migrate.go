// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp/migrate"
	v1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	"go.uber.org/zap"
)

// Migration drops the internal flag of each stored channel and logs each channel that
// held it.
var Migration = migrate.NewInstrumentedEntry(
	"v62_drop_internal",
	func(ctx context.Context, old v1.Channel, ins alamos.Instrumentation) (Channel, error) {
		if old.Internal {
			ins.L.Warn(
				"dropped the internal flag of a channel",
				zap.Stringer("key", old.Key()),
				zap.String("name", old.Name),
			)
		}
		return autoMigrateChannel(ctx, old)
	},
)
