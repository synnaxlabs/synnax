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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// LinkWriter returns the JSON options that write each channel key as the exact path of
// its channel, as snap sees it, and a key of no channel as 0. The options cache each
// path they write, so pass one value to every file of one export. They are not safe for
// concurrent use.
func (s *Service) LinkWriter(ctx context.Context, snap gorp.Snapshot) json.Options {
	w := &linkWriter{
		ctx:     ctx,
		snap:    snap,
		svc:     s,
		paths:   make(map[Key]string),
		folders: make(map[group.Key]group.Path),
	}
	return json.WithMarshalers(json.MarshalToFunc(w.write))
}

type linkWriter struct {
	ctx  context.Context
	snap gorp.Snapshot
	svc  *Service
	// paths holds the path of each key written so far, or "" for a key of no channel.
	paths   map[Key]string
	folders map[group.Key]group.Path
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
	var chs []Channel
	if err := w.svc.NewRetrieve().
		Where(MatchKeys(k)).
		SkipMissingKeys().
		Entries(&chs).
		Exec(w.ctx, w.snap); err != nil {
		return "", err
	}
	if len(chs) == 0 {
		w.paths[k] = ""
		return "", nil
	}
	ch := chs[0]
	var folders group.Path
	if ch.Group != nil {
		f, ok := w.folders[*ch.Group]
		if !ok {
			found, err := group.Paths(w.ctx, w.snap, *ch.Group)
			if err != nil {
				return "", err
			}
			f = found[*ch.Group]
			w.folders[*ch.Group] = f
		}
		folders = f
	}
	p := PathTexts([]Path{{Folders: folders, Name: ch.Name}})[0]
	w.paths[k] = p
	return p, nil
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
	var chs []Channel
	if err := r.svc.NewRetrieve().
		Where(MatchPaths(path)).
		Entries(&chs).
		Exec(r.ctx, r.snap); err != nil {
		return 0, err
	}
	if len(chs) == 0 {
		r.missing.Add(path)
		r.keys[path] = 0
		return 0, nil
	}
	k := chs[0].Key()
	r.keys[path] = k
	return k, nil
}
