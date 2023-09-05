// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package workspace

import (
	"context"
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
)

type Writer struct {
	tx  gorp.Tx
	otg ontology.Writer
}

func (w Writer) Create(
	ctx context.Context,
	ws *Workspace,
) (err error) {
	if ws.Key == uuid.Nil {
		ws.Key = uuid.New()
	}
	if err = gorp.NewCreate[uuid.UUID, Workspace]().Entry(ws).Exec(ctx, w.tx); err != nil {
		return
	}
	return err
}

func (w Writer) Delete(
	ctx context.Context,
	keys ...uuid.UUID,
) error {
	return gorp.NewDelete[uuid.UUID, Workspace]().WhereKeys(keys...).Exec(ctx, w.tx)
}
