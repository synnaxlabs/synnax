// Copyright 2023 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/confluence"
	kvx "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/version"
)

type gossipState byte

const (
	infected gossipState = iota
	recovered
)

var codec = &binary.MsgPackEncoderDecoder{}

type Operation struct {
	kvx.Change
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

func (o Operation) apply(ctx context.Context, b kvx.Writer) error {
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

func (d Digest) apply(ctx context.Context, w kvx.Writer) error {
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
		Change:      kvx.Change{Key: d.Key, Variant: d.Variant},
		Version:     d.Version,
		Leaseholder: d.Leaseholder,
	}
}

func digestKey(key []byte) (opKey []byte, err error) {
	return kvx.CompositeKey("--dig/", key)
}
