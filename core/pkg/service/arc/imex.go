// Copyright 2026 Synnax Labs, Inc.
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
	"strings"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc/versions"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/set"
)

var _ imex.ImportExporter = (*Service)(nil)

// Type implements imex.ImportExporter.
func (s *Service) Type() resource.Type { return resource.TypeArc }

// Match reports whether body is a legacy Console Arc state, which persists the graph
// inline alongside text and mode. The markers are frozen historical file shapes.
func (*Service) Match(body map[string]any) bool {
	_, hasGraph := body["graph"]
	_, hasMode := body["mode"]
	_, hasText := body["text"]
	return hasGraph && (hasMode || hasText)
}

// Decode decodes env into an Arc that its Write persists. The key on the wire is
// discarded so every import mints a new resource. An unknown envelope version is a
// path-scoped validation error.
//
// The Write ignores the parent and group in opts: the writer puts every new Arc in the
// root directory.
func (s *Service) Decode(ctx context.Context, env imex.Envelope) (imex.Write, error) {
	a, err := versions.DecodeImExEnvelope(ctx, env)
	if err != nil {
		return nil, err
	}
	// The text of an Arc exported before v5 reads a dashed channel name bare.
	bare := env.Version < versions.Latest
	return func(
		ctx context.Context,
		tx gorp.Tx,
		opts imex.ImportOptions,
	) (resource.ID, error) {
		if bare {
			if err := s.quoteDashedNames(ctx, tx, &a, opts.Retrievable); err != nil {
				return resource.ID{}, err
			}
		}
		if err := s.NewWriter(tx).Create(ctx, &a); err != nil {
			return resource.ID{}, err
		}
		return a.ResourceID(), nil
	}, nil
}

// quoteDashedNames quotes each dashed channel name in the source of a that a channel
// holds. A dashed name that no channel holds stays, as the v63 lexer reads it. A
// channel
// that a non-nil retrievable refuses counts as absent, so the result shows no hidden
// channel.
func (s *Service) quoteDashedNames(
	ctx context.Context,
	tx gorp.Tx,
	a *Arc,
	retrievable func(resource.ID) bool,
) error {
	b := a.Base()
	var dashed []channel.Name
	RenameChannels(&b.Source, func(name string) (string, bool) {
		if strings.IndexByte(name, '-') >= 0 {
			dashed = append(dashed, name)
		}
		return "", false
	})
	if len(dashed) == 0 {
		return nil
	}
	var held []channel.Channel
	if err := s.cfg.Channel.NewRetrieve().
		Where(channel.MatchNames(dashed...)).
		Entries(&held).
		Exec(ctx, tx); err != nil {
		return err
	}
	names := make(set.Set[string], len(held))
	for _, c := range held {
		if retrievable == nil || retrievable(c.ResourceID()) {
			names.Add(c.Name)
		}
	}
	if RenameChannels(&b.Source, func(name string) (string, bool) {
		return name, names.Contains(name)
	}) {
		a.SetBase(b)
	}
	return nil
}
