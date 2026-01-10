// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv

import (
	"context"

	"github.com/synnaxlabs/aspen/node"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/confluence"
	xkv "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/version"
)

type gossipState byte

const (
	infected gossipState = iota
	recovered
)

// codec used to be implemented by a gob codec, but we want to switch to msgpack.
// Instead, we will use a fallback codec that tries msgpack to decode first, then gob.
var codec = binary.NewDecodeFallbackCodec(&binary.MsgPackCodec{}, &binary.GobCodec{})

type Operation struct {
	xkv.Change
	Version     version.Counter
	Leaseholder node.Key
	state       gossipState
}

func (o Operation) Digest() Digest {
	return Digest{
		Key:         o.Key,
		Version:     o.Version,
		Leaseholder: o.Leaseholder,
		Variant:     o.Variant,
	}
}

func (o Operation) apply(ctx context.Context, b xkv.Writer) error {
	if o.Variant == change.Delete {
		return b.Delete(ctx, o.Key)
	}
	return b.Set(ctx, o.Key, o.Value)
}

type Digest struct {
	Key         []byte
	Variant     change.Variant
	Version     version.Counter
	Leaseholder node.Key
}

func (d Digest) apply(ctx context.Context, w xkv.Writer) error {
	key, err := digestKey(d.Key)
	if err != nil {
		return err
	}
	b, err := codec.Encode(ctx, d)
	if err != nil {
		return err
	}
	return w.Set(ctx, key, b)
}

type Digests []Digest

func (d Digests) toRequest(ctx context.Context) TxRequest {
	txr := TxRequest{Context: ctx, Operations: make([]Operation, len(d))}
	for i, d := range d {
		txr.Operations[i] = d.Operation()
	}
	return txr
}

type (
	segment = confluence.Segment[TxRequest, TxRequest]
	source  = confluence.Source[TxRequest]
	sink    = confluence.Sink[TxRequest]
)

func (d Digest) Operation() Operation {
	return Operation{
		Change:      xkv.Change{Key: d.Key, Variant: d.Variant},
		Version:     d.Version,
		Leaseholder: d.Leaseholder,
	}
}

const digestPrefix = "--dig/"

func digestKey(key []byte) (opKey []byte, err error) {
	return xkv.CompositeKey(digestPrefix, key)
}
