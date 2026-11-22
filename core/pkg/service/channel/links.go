// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"slices"
	"strings"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// LinkWriter returns the JSON options that write each channel key as the exact path of
// its channel, as snap sees it, and a key of no channel as 0. The options cache each
// path they write, so pass one value to every file of one export. They are not safe for
// concurrent use.
func (s *Service) LinkWriter(ctx context.Context, snap gorp.Snapshot) json.Options {
	w := &linkWriter{
		ctx:      ctx,
		snap:     snap,
		svc:      s,
		paths:    make(map[Key]string),
		prefixes: make(map[group.Key]string),
	}
	return json.WithMarshalers(json.MarshalToFunc(w.write))
}

type linkWriter struct {
	ctx  context.Context
	snap gorp.Snapshot
	svc  *Service
	// paths holds the path of each key written so far, or "" for a key of no channel.
	paths map[Key]string
	// prefixes holds the path text of each folder read so far, with a trailing
	// separator.
	prefixes map[group.Key]string
}

func (w *linkWriter) write(enc *jsontext.Encoder, k Key) error {
	p, err := w.path(k)
	if err != nil {
		return err
	}
	if p == "" {
		return enc.WriteToken(jsontext.Uint(0))
	}
	return enc.WriteToken(jsontext.String(p))
}

func (w *linkWriter) path(k Key) (string, error) {
	if k == 0 {
		return "", nil
	}
	if p, ok := w.paths[k]; ok {
		return p, nil
	}
	sibling, ok, err := w.svc.indexes.sibling.Value(w.ctx, w.snap, k)
	if err != nil {
		return "", err
	}
	if !ok {
		w.paths[k] = ""
		return "", nil
	}
	p := sibling.Name
	if sibling.Parent != (group.Key{}) {
		prefix, err := w.prefix(sibling.Parent)
		if err != nil {
			return "", err
		}
		p = prefix + p
	}
	w.paths[k] = p
	return p, nil
}

// prefix returns the path text of folder with a trailing separator. It builds on the
// prefix of the folder's parent, so sibling folders share their ancestors' lookups.
func (w *linkWriter) prefix(folder group.Key) (string, error) {
	if prefix, ok := w.prefixes[folder]; ok {
		return prefix, nil
	}
	sibling, ok, err := w.svc.indexes.sibling.ValueOf(
		w.ctx,
		w.snap,
		gorp.NewTarget(string(resource.TypeGroup), folder),
	)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errors.Wrapf(query.ErrNotFound, "folder %s", folder)
	}
	var parent string
	if sibling.Parent != (group.Key{}) {
		if parent, err = w.prefix(sibling.Parent); err != nil {
			return "", err
		}
	}
	prefix := parent + sibling.Name + string(validate.PathSeparator)
	w.prefixes[folder] = prefix
	return prefix, nil
}

// LinkReader decodes the channel links of imported files. A JSON number is a raw key,
// as every file of core/v0.62.0 and older holds. A JSON string is the exact path of a
// channel, which it resolves once against the snapshot it was made with. A path that
// matches no channel decodes as 0, and Err then fails. A LinkReader is not safe for
// concurrent use.
type LinkReader struct {
	ctx     context.Context
	snap    gorp.Snapshot
	svc     *Service
	keys    map[string]Key
	missing set.Set[string]
}

// NewLinkReader returns a LinkReader that resolves paths against snap. Pass one value
// to every file of one import, so that Err lists every path that matches nothing.
func (s *Service) NewLinkReader(ctx context.Context, snap gorp.Snapshot) *LinkReader {
	return &LinkReader{
		ctx:     ctx,
		snap:    snap,
		svc:     s,
		keys:    make(map[string]Key),
		missing: make(set.Set[string]),
	}
}

// Options returns the JSON options that decode each channel key through r.
func (r *LinkReader) Options() json.Options {
	return json.WithUnmarshalers(json.UnmarshalFromFunc(r.read))
}

// Err returns a validation error that lists each path that matched no channel, sorted,
// or nil when every path matched one.
func (r *LinkReader) Err() error {
	if len(r.missing) == 0 {
		return nil
	}
	paths := r.missing.Slice()
	slices.Sort(paths)
	return errors.Wrapf(
		validate.ErrValidation,
		"no channel is at %s",
		strings.Join(paths, ", "),
	)
}

func (r *LinkReader) read(dec *jsontext.Decoder, k *Key) error {
	if dec.PeekKind() != '"' {
		return errors.ErrUnsupported
	}
	tok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	*k, err = r.resolve(tok.String())
	return err
}

func (r *LinkReader) resolve(path string) (Key, error) {
	if k, ok := r.keys[path]; ok {
		return k, nil
	}
	k, ok, err := r.lookup(path)
	if err != nil {
		return 0, err
	}
	if !ok {
		r.missing.Add(path)
	}
	r.keys[path] = k
	return k, nil
}

func (r *LinkReader) lookup(path string) (Key, bool, error) {
	sibling, ok, err := Retrieve{indexes: r.svc.indexes}.siblingAt(
		r.ctx,
		r.snap,
		validate.NormalizeName(path),
	)
	if err != nil || !ok {
		return 0, false, err
	}
	return r.svc.indexes.sibling.Key(r.ctx, r.snap, sibling)
}
