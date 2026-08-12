// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import (
	"context"
	"maps"
	"slices"

	"github.com/synnaxlabs/oracle/resolution"
)

// Annotate stamps chain-derived @go version expressions onto the live table: every type
// a chain's current file enumerates receives its chain's current version. The version
// files are the authority; the injected expressions are how the generators read it.
// Every other version-owned fact (marshal, migrate, imex) reaches the table textually
// through the merged live file. A type that still hand-declares a version must agree
// with its chain.
func (r *Resolver) Annotate(ctx context.Context, table *resolution.Table) error {
	for _, livePath := range slices.Sorted(maps.Keys(r.chains)) {
		chain := r.chains[livePath]
		current := chain.Current()
		surf, err := r.Surface(ctx, livePath, current)
		if err != nil {
			return err
		}
		for i := range table.Types {
			t := &table.Types[i]
			if t.Namespace != chain.Resource ||
				t.FilePath != livePath+".oracle" {
				continue
			}
			if _, member := surf[t.Name]; !member {
				continue
			}
			injectVersion(t, current)
		}
	}
	return nil
}

// injectVersion adds a @go version expression to the type, cloning the shared domain
// maps first.
func injectVersion(t *resolution.Type, version int) {
	domains := maps.Clone(t.Domains)
	if domains == nil {
		domains = make(map[string]resolution.Domain)
	}
	dom := domains["go"]
	dom.Name = "go"
	values := []resolution.ExpressionValue{{
		Kind:     resolution.ValueKindInt,
		IntValue: int64(version),
	}}
	dom.Expressions = append(
		slices.Clone(dom.Expressions),
		resolution.Expression{Name: "version", Values: values},
	)
	domains["go"] = dom
	t.Domains = domains
}
