// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package invite

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
)

// Writer deletes invites in one transaction. It makes no identity check:
// the caller authorizes the change.
type Writer struct {
	svc *Service
	tx  gorp.Tx
}

// Delete removes the invites with the given keys. Unknown keys are ignored.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.svc.table.NewDelete().
		Where(gorp.MatchKeys[Key, Invite](keys...)).
		Exec(ctx, w.tx)
}

// DeleteFor removes the invites of the given subjects.
func (w Writer) DeleteFor(ctx context.Context, subjects ...ontology.ID) error {
	return w.svc.table.NewDelete().
		Where(MatchSubjects(subjects...)(w.svc.NewRetrieve())).
		Exec(ctx, w.tx)
}
