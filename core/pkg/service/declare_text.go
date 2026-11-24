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
// start that stops first runs the pass again. Nothing may compile that text between a
// Declare and the pass.
func passDeclaredRenames(
	ctx context.Context,
	db *gorp.DB,
	channels *channel.Service,
) error {
	return db.WithTx(ctx, func(tx gorp.Tx) error {
		stored, err := channels.NewWriter(tx).TakeTextRenames(ctx)
		if err != nil || len(stored) == 0 {
			return err
		}
		keys := make([]channel.Key, len(stored))
		for i, r := range stored {
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
		// A deleted channel keeps the tokens of its old path.
		var renames []declaredRename
		for _, r := range stored {
			if to, ok := paths[r.Key]; ok {
				renames = append(renames, declaredRename{from: r.From, to: to})
			}
		}
		cfg := parser.Config{AllowDashedNames: resolver.AllowDashedNames(channels)}
		if err = renameCalculations(ctx, tx, renames, cfg); err != nil {
			return err
		}
		return renameArcs(ctx, tx, renames, cfg)
	})
}

// declaredRename is a rename that the pass applies: from the old path to the new one.
type declaredRename struct{ from, to string }

// rewriteSource returns source with the tokens of each rename rewritten. block reads
// source as the body of a function.
func rewriteSource(
	source string,
	block bool,
	renames []declaredRename,
	cfg parser.Config,
) string {
	for _, r := range renames {
		if next, ok := text.RenameChannel(source, block, r.from, r.to, cfg); ok {
			source = next
		}
	}
	return source
}

// renameCalculations rewrites the expression of each calculation.
func renameCalculations(
	ctx context.Context,
	tx gorp.Tx,
	renames []declaredRename,
	cfg parser.Config,
) error {
	var calcs []calculation.Calculation
	if err := gorp.NewRetrieve[calculation.Key, calculation.Calculation]().
		Entries(&calcs).
		Exec(ctx, tx); err != nil {
		return err
	}
	var changed []calculation.Calculation
	for _, c := range calcs {
		if e := rewriteSource(c.Expression, true, renames, cfg); e != c.Expression {
			c.Expression = e
			changed = append(changed, c)
		}
	}
	return gorp.WrapWriter[calculation.Key, calculation.Calculation](tx).
		Set(ctx, changed...)
}

// renameArcs rewrites the text of each Arc through its op-log, and the body of each
// function of each Arc graph.
func renameArcs(
	ctx context.Context,
	tx gorp.Tx,
	renames []declaredRename,
	cfg parser.Config,
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
		if rewriteArcSource(&b.Source, renames, cfg) {
			a.SetBase(b)
			changed = append(changed, a)
		}
	}
	return gorp.WrapWriter[arc.Key, arc.Arc](tx).Set(ctx, changed...)
}

// rewriteArcSource rewrites src, and reports whether it changed. A function body keeps
// its braces.
func rewriteArcSource(
	src *arc.Source,
	renames []declaredRename,
	cfg parser.Config,
) bool {
	changed := false
	switch v := src.Variant.(type) {
	case arc.TextSource:
		for _, r := range renames {
			doc, ok := text.RenameChannelInDocument(v.Text.Doc, r.from, r.to, cfg)
			// A rename inserts the characters of each new token.
			if ok && len(doc.Inserts) != len(v.Text.Doc.Inserts) {
				v.Text.Doc = doc
				src.Variant, changed = v, true
			}
		}
	case arc.GraphSource:
		for i, f := range v.Graph.Functions {
			body := strings.TrimSpace(f.Body.Raw)
			if !strings.HasPrefix(body, "{") || !strings.HasSuffix(body, "}") {
				continue
			}
			inner := body[1 : len(body)-1]
			if renamed := rewriteSource(inner, true, renames, cfg); renamed != inner {
				v.Graph.Functions[i].Body.Raw = "{" + renamed + "}"
				src.Variant, changed = v, true
			}
		}
	}
	return changed
}
