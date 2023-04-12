// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"
	"github.com/synnaxlabs/x/gorp"
)

type Writer struct {
	proxy *leaseProxy
	tx    gorp.Tx
}

func (w Writer) Create(ctx context.Context, c *Channel) error {
	channels := []Channel{*c}
	err := w.proxy.create(ctx, w.tx, &channels)
	if err != nil {
		return err
	}
	*c = channels[0]
	return nil
}

func (w Writer) CreateMany(ctx context.Context, channels *[]Channel) error {
	return w.proxy.create(ctx, w.tx, channels)
}
