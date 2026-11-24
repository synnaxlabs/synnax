// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http

import (
	"context"

	"github.com/synnaxlabs/gorp"
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
)

func reconcileRead(_ context.Context, _ gorp.Tx, c *ReadConfig) error {
	v5.RepairReadConfig(c)
	return nil
}

func reconcileWrite(_ context.Context, _ gorp.Tx, c *WriteConfig) error {
	v5.RepairWriteConfig(c)
	return nil
}
