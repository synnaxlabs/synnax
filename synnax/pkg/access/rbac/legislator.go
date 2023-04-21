// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
)

type Legislator struct {
	DB *gorp.DB
}

func (l *Legislator) Create(txn gorp.Tx, p Policy) error {
	return gorp.NewCreate[string, Policy]().Entry(&p).Exec(context.TODO(), txn)
}

func (l *Legislator) Retrieve(ctx context.Context, subject, object ontology.ID) ([]Policy, error) {
	var p []Policy
	return p, gorp.NewRetrieve[string, Policy]().
		WhereKeys(NewPolicyKey(subject, object)).
		Entries(&p).
		Exec(ctx, l.DB)
}
