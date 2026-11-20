// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service

import (
	"context"
	"slices"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// newCalculationGrantMigration returns the migration that gives each stored policy that
// is not built in, and grants actions on channels, the same actions on their
// calculations: a grant on the channel type also grants on the calculation type, and a
// grant on a channel with a calculation also grants on that calculation. Under v0.62 a
// calculation was part of its channel. It runs after v63_create_calculations.
func newCalculationGrantMigration(
	policies *policy.Service,
	calculations *calculation.Service,
) gorp.Migration {
	return migrate.New(
		"v63_grant_calculations",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			var stored []policy.Policy
			if err := policies.NewRetrieve().
				Entries(&stored).
				Exec(ctx, tx); err != nil {
				return err
			}
			var calcs []calculation.Calculation
			if err := calculations.NewRetrieve().
				Entries(&calcs).
				Exec(ctx, tx); err != nil {
				return err
			}
			byOutput := make(map[string]resource.ID, len(calcs))
			for _, c := range calcs {
				byOutput[c.Output.String()] = calculation.ResourceID(c.Key)
			}
			w := policies.NewWriter(tx, true)
			for _, p := range stored {
				if p.Internal {
					continue
				}
				granted := calculationGrants(p.Objects, byOutput)
				if len(granted) == 0 {
					continue
				}
				p.Objects = append(p.Objects, granted...)
				if err := w.Delete(ctx, p.Key); err != nil {
					return err
				}
				if err := w.Create(ctx, &p); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

// calculationGrants returns the calculation objects that a policy on objects gains,
// leaving out those it already holds.
func calculationGrants(
	objects []resource.ID,
	byOutput map[string]resource.ID,
) []resource.ID {
	var granted []resource.ID
	add := func(id resource.ID) {
		if !slices.Contains(objects, id) && !slices.Contains(granted, id) {
			granted = append(granted, id)
		}
	}
	for _, o := range objects {
		if o.Type != resource.TypeChannel {
			continue
		}
		if o.Key == "" {
			add(resource.ID{Type: resource.TypeCalculation})
		} else if id, ok := byOutput[o.Key]; ok {
			add(id)
		}
	}
	return granted
}
