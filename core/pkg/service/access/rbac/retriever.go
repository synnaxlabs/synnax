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
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
)

type Retriever struct {
	baseTx gorp.Tx
	gorp   gorp.Retrieve[uuid.UUID, Policy]
}

func (r Retriever) WhereSubjects(subjects ...ontology.ID) Retriever {
	r.gorp = r.gorp.Where(func(ctx gorp.Context, p *Policy) (bool, error) {
		for _, subject := range p.Subjects {
			if lo.Contains(subjects, subject) {
				return true, nil
			}
			if subject.IsType() {
				for _, s := range subjects {
					if s.Type == subject.Type {
						return true, nil
					}
				}
			}
		}
		return false, nil
	})
	return r
}

func (r Retriever) WhereKeys(keys ...uuid.UUID) Retriever {
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}

func (r Retriever) Exec(ctx context.Context, tx gorp.Tx) error {
	tx = gorp.OverrideTx(r.baseTx, tx)
	return r.gorp.Exec(ctx, tx)
}

func (r Retriever) Entry(p *Policy) Retriever {
	r.gorp = r.gorp.Entry(p)
	return r
}

func (r Retriever) Entries(ps *[]Policy) Retriever {
	r.gorp = r.gorp.Entries(ps)
	return r
}
