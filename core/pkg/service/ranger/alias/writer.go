// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alias

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

// Writer is used to create and delete aliases.
type Writer struct {
	tx        gorp.Tx
	otg       *ontology.Ontology
	otgWriter ontology.Writer
}

// Set sets an alias for the given channel on the specified range.
func (w Writer) Set(ctx context.Context, rng uuid.UUID, ch channel.Key, alias string) error {
	exists, err := gorp.NewRetrieve[channel.Key, channel.Channel]().WhereKeys(ch).Exists(ctx, w.tx)
	if err != nil {
		return err
	}
	if !exists {
		return errors.Wrapf(query.NotFound, "[alias] - cannot alias non-existent channel %s", ch)
	}
	if err := gorp.NewCreate[string, Alias]().
		Entry(&Alias{Range: rng, Channel: ch, Alias: alias}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otgWriter.DefineResource(ctx, OntologyID(rng, ch))
}

// Delete deletes the alias for the given channel on the specified range.
// Delete is idempotent and will not return an error if the alias does not exist.
func (w Writer) Delete(ctx context.Context, rng uuid.UUID, ch channel.Key) error {
	return gorp.
		NewDelete[string, Alias]().
		WhereKeys(Alias{Range: rng, Channel: ch}.GorpKey()).
		Exec(ctx, w.tx)
}
