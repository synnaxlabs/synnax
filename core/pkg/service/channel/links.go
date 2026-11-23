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
// its channel, as snap sees it, and a key of no channel as 0. A channel that readable
// refuses, or that sits in a folder readable refuses, writes its raw key, so the file
// names no record that the exporting subject cannot retrieve. The options cache each
// link they write, so pass one value to every file of one export. They are not safe for
// concurrent use.
func (s *Service) LinkWriter(
	ctx context.Context,
	snap gorp.Snapshot,
	readable func(resource.ID) bool,
) json.Options {
	w := &linkWriter{
		ctx:      ctx,
		snap:     snap,
		svc:      s,
		readable: readable,
		links:    make(map[Key]link),
		prefixes: make(map[group.Key]link),
	}
	return json.WithMarshalers(json.MarshalToFunc(w.write))
}

// link is how a channel or a folder appears in an exported file: its path text, or
// hidden when the exporting subject cannot retrieve it or a folder above it.
type link struct {
	text   string
	hidden bool
}

type linkWriter struct {
	ctx      context.Context
	snap     gorp.Snapshot
	svc      *Service
	readable func(resource.ID) bool
	// links holds the link of each key written so far. A key of no channel has an
	// empty text.
	links map[Key]link
	// prefixes holds the path text of each folder read so far, with a trailing
	// separator.
	prefixes map[group.Key]link
}

func (w *linkWriter) write(enc *jsontext.Encoder, k Key) error {
	l, err := w.link(k)
	if err != nil {
		return err
	}
	switch {
	case l.hidden:
		return enc.WriteToken(jsontext.Uint(uint64(k)))
	case l.text == "":
		return enc.WriteToken(jsontext.Uint(0))
	default:
		return enc.WriteToken(jsontext.String(l.text))
	}
}

func (w *linkWriter) link(k Key) (link, error) {
	if k == 0 {
		return link{}, nil
	}
	if l, ok := w.links[k]; ok {
		return l, nil
	}
	sibling, ok, err := w.svc.indexes.sibling.Value(w.ctx, w.snap, k)
	if err != nil {
		return link{}, err
	}
	l := link{text: sibling.Name, hidden: !w.readable(ResourceID(k))}
	switch {
	case !ok:
		l = link{}
	case !l.hidden && sibling.Parent != (group.Key{}):
		prefix, err := w.prefix(sibling.Parent)
		if err != nil {
			return link{}, err
		}
		l = link{text: prefix.text + l.text, hidden: prefix.hidden}
	}
	w.links[k] = l
	return l, nil
}

// prefix returns the path text of folder with a trailing separator. It builds on the
// prefix of the folder's parent, so sibling folders share their ancestors' lookups.
func (w *linkWriter) prefix(folder group.Key) (link, error) {
	if prefix, ok := w.prefixes[folder]; ok {
		return prefix, nil
	}
	sibling, ok, err := w.svc.indexes.sibling.ValueOf(
		w.ctx,
		w.snap,
		gorp.NewTarget(string(resource.TypeGroup), folder),
	)
	if err != nil {
		return link{}, err
	}
	if !ok {
		return link{}, errors.Wrapf(query.ErrNotFound, "folder %s", folder)
	}
	prefix := link{
		text:   sibling.Name + string(validate.PathSeparator),
		hidden: !w.readable(group.ResourceID(folder)),
	}
	if !prefix.hidden && sibling.Parent != (group.Key{}) {
		parent, err := w.prefix(sibling.Parent)
		if err != nil {
			return link{}, err
		}
		prefix = link{text: parent.text + prefix.text, hidden: parent.hidden}
	}
	w.prefixes[folder] = prefix
	return prefix, nil
}

// LinkReader decodes the channel links of imported files. A JSON number is a raw key,
// as every file of core/v0.62.0 and older holds. A JSON string is the exact path of a
// channel, which it resolves once against the snapshot it was made with: the committed
// state and the snapshot's own changes, not a commit in flight. An import resolves its
// links before it writes, so a file cannot link a channel that the same import
// creates. A path that matches no channel, or that passes a folder or names a channel
// that the importer cannot retrieve, decodes as 0, and Err then fails, so an import
// never shows that a hidden path exists. A LinkReader is not safe for concurrent use.
type LinkReader struct {
	ctx      context.Context
	snap     gorp.Snapshot
	svc      *Service
	readable func(resource.ID) bool
	keys     map[string]Key
	missing  set.Set[string]
}

// NewLinkReader returns a LinkReader that resolves paths against snap through the
// folders and channels that readable allows. Pass one value to every file of one
// import, so that Err lists every path that matches nothing.
func (s *Service) NewLinkReader(
	ctx context.Context,
	snap gorp.Snapshot,
	readable func(resource.ID) bool,
) *LinkReader {
	return &LinkReader{
		ctx:      ctx,
		snap:     snap,
		svc:      s,
		readable: readable,
		keys:     make(map[string]Key),
		missing:  make(set.Set[string]),
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
	sibling, ok, err := Retrieve{
		indexes:       r.svc.indexes,
		folderAllowed: r.readable,
	}.siblingAt(r.ctx, r.snap, validate.NormalizeName(path))
	if err != nil || !ok {
		return 0, false, err
	}
	k, ok, err := r.svc.indexes.sibling.Key(r.ctx, r.snap, sibling)
	if err != nil || !ok || !r.readable(ResourceID(k)) {
		return 0, false, err
	}
	return k, true, nil
}
