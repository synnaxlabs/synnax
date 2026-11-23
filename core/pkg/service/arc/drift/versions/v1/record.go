// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
)

var _ gorp.Entry[Key] = Record{}

func (r Record) GorpKey() Key { return r.Task }

// Migration lifts each stored drift record to v1, with no deployed channels.
var Migration = migrate.NewEntry("v63_drift_compile_keys", autoMigrateRecord)
