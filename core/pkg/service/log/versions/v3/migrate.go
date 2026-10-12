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

	v2 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v2"
	"github.com/synnaxlabs/x/gorp"
)

// MigrateLog lifts a v2 log to v3 with the default range, a rolling 7d window.
func MigrateLog(ctx context.Context, old v2.Log) (Log, error) {
	out, err := autoMigrateLog(ctx, old)
	if err != nil {
		return Log{}, err
	}
	out.ApplyDefaults()
	return out, nil
}

// Migration lifts stored logs from v2 to v3.
var Migration = gorp.NewEntryMigration("v60_log_range", MigrateLog)
