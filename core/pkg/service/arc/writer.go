// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/arc/core"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
)

// Writer is used to create, update, and delete arcs within Synnax. The writer
// executes all operations within the transaction provided to the Service.NewWriter
// method. If no transaction is provided, the writer will execute operations directly
// on the database.
type Writer struct {
	tx     gorp.Tx
	otg    ontology.Writer
	status status.Writer[core.StatusDetails]
}

// Create creates the given Arc. If the Arc does not have a key,
// a new key will be generated.
func (w Writer) Create(
	ctx context.Context,
	c *Arc,
) error {
	var (
		exists bool
		err    error
	)
	if c.Key == uuid.Nil {
		c.Key = uuid.New()
	} else {
		exists, err = gorp.NewRetrieve[uuid.UUID, Arc]().WhereKeys(c.Key).Exists(ctx, w.tx)
		if err != nil {
			return err
		}
	}
	if err = gorp.NewCreate[uuid.UUID, Arc]().Entry(c).Exec(ctx, w.tx); err != nil {
		return err
	}
	otgID := OntologyID(c.Key)
	if !exists {
		if err = w.otg.DefineResource(ctx, otgID); err != nil {
			return err
		}
	}

	return w.status.SetWithParent(ctx, &status.Status[core.StatusDetails]{
		Name:    fmt.Sprintf("%s Status", c.Name),
		Key:     c.Key.String(),
		Variant: xstatus.LoadingVariant,
		Message: "Deploying",
		Time:    telem.Now(),
		Details: core.StatusDetails{Running: false},
	}, otgID)
}

// Delete deletes the arcs with the given keys.
func (w Writer) Delete(
	ctx context.Context,
	keys ...uuid.UUID,
) (err error) {
	if err = gorp.NewDelete[uuid.UUID, Arc]().WhereKeys(keys...).Exec(ctx, w.tx); err != nil {
		return
	}
	statusKeys := lo.Map(keys, func(k uuid.UUID, _ int) string { return k.String() })
	if err = w.status.DeleteMany(ctx, statusKeys...); err != nil {
		return err
	}
	for _, key := range keys {
		if err = w.otg.DeleteResource(ctx, OntologyID(key)); err != nil {
			return
		}
	}
	return
}
