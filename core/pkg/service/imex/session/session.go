// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package session exports and imports the files of one request, with the channel links
// of each file as paths. One Export or Import shares its channel lookups across every
// file of the request.
package session

import (
	"context"
	"encoding/json/v2"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Export exports the records of one snapshot. It is not safe for concurrent use.
type Export struct {
	ctx   context.Context
	snap  gorp.Snapshot
	imex  *imex.Service
	links json.Options
}

// NewExport returns an Export of the records of snap through registry. A channel that
// readable refuses, or one in a folder that readable refuses, is written as its raw
// key, so the file never names it.
func NewExport(
	ctx context.Context,
	registry *imex.Service,
	channels *channel.Service,
	snap gorp.Snapshot,
	readable func(resource.ID) bool,
) *Export {
	return &Export{
		ctx:   ctx,
		snap:  snap,
		imex:  registry,
		links: channels.LinkWriter(ctx, snap, readable),
	}
}

// Export returns the envelope of the record id, whose channel links encode as paths.
func (e *Export) Export(id resource.ID) (imex.Envelope, error) {
	env, err := e.imex.Export(e.ctx, e.snap, id)
	if err != nil {
		return imex.Envelope{}, err
	}
	return env.WithEncodeOptions(e.links), nil
}

// Import decodes the files of one import. It is not safe for concurrent use.
type Import struct {
	ctx   context.Context
	imex  *imex.Service
	links *channel.LinkReader
}

// NewImport returns an Import that decodes through registry and resolves channel paths
// against snap through the folders and channels that readable allows.
func NewImport(
	ctx context.Context,
	registry *imex.Service,
	channels *channel.Service,
	snap gorp.Snapshot,
	readable func(resource.ID) bool,
) *Import {
	return &Import{
		ctx:   ctx,
		imex:  registry,
		links: channels.NewLinkReader(ctx, snap, readable),
	}
}

// Decode decodes env as imex.Service.Decode does, with its channel paths resolved.
func (i *Import) Decode(env imex.Envelope, fileName string) (imex.Write, error) {
	return i.imex.Decode(i.ctx, env.WithDecodeOptions(i.links.Options()), fileName)
}

// Err returns a validation error that lists each channel path of a decoded file that
// matched no channel the importer can retrieve, or nil when every path matched one.
func (i *Import) Err() error { return i.links.Err() }
