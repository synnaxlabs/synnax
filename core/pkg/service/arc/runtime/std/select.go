// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package std

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/stage"
)

type selectStage struct{ base }

func (s *selectStage) Next(ctx context.Context, value stage.Value) {
	if value.Value == 0 {
		s.outputHandler(ctx, value)
	} else {
		s.outputHandler(ctx, value)
	}
}
