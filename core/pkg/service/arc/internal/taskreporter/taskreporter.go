// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package taskreporter exports the Reporter callback used by Arc stdlib
// modules to write task-level status updates. Currently only used by the
// status module.
//
// Lives under internal/ because only the arc runtime constructs Reporters
// today; if non-arc packages ever need this shape, move the file to
// core/pkg/service/arc/taskreporter.
package taskreporter

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/service/status"
)

// Reporter writes a task-level status update from inside an stdlib module.
type Reporter func(ctx context.Context, variant status.Variant, message string)
