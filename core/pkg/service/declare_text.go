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
	"strings"

	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel/resolver"
)

// passDeclaredRenames runs the text pass of each rename that Declare stored
// (channel.Writer.TakeTextRenames): each token in the text of a calculation or an Arc
// that names the old path becomes a token of the channel's path now, whatever channel
// holds the old path by then. It deletes the renames in the same transaction, so a
// start that stops first runs the pass again. It must run after every Declare of the
// start and before anything compiles that text again.
func passDeclaredRenames(
	ctx context.Context,
	db *gorp.DB,
	channels *channel.Service,
) error {
	return db.WithTx(ctx, func(tx gorp.Tx) error {
		renames, err := channels.NewWriter(tx).TakeTextRenames(ctx)
		if err != nil || len(renames) == 0 {
			return err
		}
		keys := make([]channel.Key, len(renames))
		for i, r := range renames {
			keys[i] = r.Key
		}
		var renamed []channel.Channel
		if err = channels.NewRetrieve().
			Where(channel.MatchKeys(keys...)).
			SkipMissingKeys().
			Entries(&renamed).
			Exec(ctx, tx); err != nil {
			return err
		}
		paths := make(map[channel.Key]string, len(renamed))
		for _, ch := range renamed {
			paths[ch.Key()] = ch.Name
		}
		cfg := parser.Config{AllowDashedNames: resolver.AllowDashedNames(channels)}
		// rename returns source with each rename's tokens rewritten. A deleted channel
		// keeps the tokens of its old path.
		rename := func(source string, block bool) string {
			for _, r := range renames {
				to, ok := paths[r.Key]
				if !ok {
					continue
				}
				if next, ok := text.RenameChannel(source, block, r.From, to, cfg); ok {
					source = next
				}
			}
			return source
		}
		if err = renameCalculations(ctx, tx, rename); err != nil {
			return err
		}
		return renameArcs(ctx, tx, rename)
	})
}

// renameCalculations writes the expression of each calculation through rename.
func renameCalculations(
	ctx context.Context,
	tx gorp.Tx,
	rename func(source string, block bool) string,
) error {
	var calcs []calculation.Calculation
	if err := gorp.NewRetrieve[calculation.Key, calculation.Calculation]().
		Entries(&calcs).
		Exec(ctx, tx); err != nil {
		return err
	}
	var changed []calculation.Calculation
	for _, c := range calcs {
		if e := rename(c.Expression, true); e != c.Expression {
			c.Expression = e
			changed = append(changed, c)
		}
	}
	return gorp.WrapWriter[calculation.Key, calculation.Calculation](tx).
		Set(ctx, changed...)
}

// renameArcs writes the text of each Arc, and the body of each function of each Arc
// graph, through rename.
func renameArcs(
	ctx context.Context,
	tx gorp.Tx,
	rename func(source string, block bool) string,
) error {
	var arcs []arc.Arc
	if err := gorp.NewRetrieve[arc.Key, arc.Arc]().
		Entries(&arcs).
		Exec(ctx, tx); err != nil {
		return err
	}
	var changed []arc.Arc
	for _, a := range arcs {
		b := a.Base()
		if renameArcSource(&b.Source, rename) {
			a.SetBase(b)
			changed = append(changed, a)
		}
	}
	return gorp.WrapWriter[arc.Key, arc.Arc](tx).Set(ctx, changed...)
}

// renameArcSource writes src through rename, and reports whether it changed. A
// function body keeps its braces.
func renameArcSource(
	src *arc.Source,
	rename func(source string, block bool) string,
) bool {
	changed := false
	switch v := src.Variant.(type) {
	case arc.TextSource:
		raw := v.Text.Materialize().Raw
		if renamed := rename(raw, false); renamed != raw {
			v.Text = text.Text{Doc: text.Create(renamed)}
			src.Variant, changed = v, true
		}
	case arc.GraphSource:
		for i, f := range v.Graph.Functions {
			body := strings.TrimSpace(f.Body.Raw)
			if !strings.HasPrefix(body, "{") || !strings.HasSuffix(body, "}") {
				continue
			}
			inner := body[1 : len(body)-1]
			if renamed := rename(inner, true); renamed != inner {
				v.Graph.Functions[i].Body.Raw = "{" + renamed + "}"
				src.Variant, changed = v, true
			}
		}
	}
	return changed
}
