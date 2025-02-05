// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rbac

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
)

type Writer struct {
	tx gorp.Tx
}

func (w Writer) Create(
	ctx context.Context,
	p *Policy,
) error {
	if p.Key == uuid.Nil {
		p.Key = uuid.New()
	}
	return gorp.NewCreate[uuid.UUID, Policy]().Entry(p).Exec(ctx, w.tx)
}

func (w Writer) Delete(
	ctx context.Context,
	keys ...uuid.UUID,
) error {
	return gorp.NewDelete[uuid.UUID, Policy]().WhereKeys(keys...).Exec(ctx, w.tx)
}
