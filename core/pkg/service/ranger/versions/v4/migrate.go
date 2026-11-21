// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4

import (
	"context"

	v1 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v1"
	"github.com/synnaxlabs/x/errors"
)

// MigrateRef copies a v1 reference. A key reference starts with no custom T0, and no
// reference has a color until a line plot write gives it one.
func MigrateRef(_ context.Context, old v1.Ref) (Ref, error) {
	switch v := old.Variant.(type) {
	case v1.KeyRef:
		return Ref{Variant: KeyRef{Key: v.Key}}, nil
	case v1.WindowRef:
		return Ref{Variant: WindowRef{Window: v.Window}}, nil
	}
	return Ref{}, errors.Newf("unknown range reference variant %T", old.Variant)
}
