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
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/confluence"
	kvx "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/version"
	"github.com/synnaxlabs/aspen/internal/node"
)

type Variant uint32

const (
	Set Variant = iota
	Delete
)

type gossipState byte

const (
	infected gossipState = iota
	recovered
)

var ecd = &binary.GobEncoderDecoder{}

type Operation struct {
	Key         []byte
	Value       []byte
	Variant     Variant
	Version     version.Counter
	Leaseholder node.ID
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

func (o Operation) apply(b kvx.Writer) error {
	if o.Variant == Delete {
		return b.Delete(o.Key)
	} else {
		return b.Set(o.Key, o.Value)
	}
}

type Digest struct {
	Key         []byte
	Version     version.Counter
	Leaseholder node.ID
	Variant     Variant
}

func (d Digest) apply(w kvx.Writer) error {
	key, err := digestKey(d.Key)
	if err != nil {
		return err
	}
	b, err := ecd.Encode(d)
	if err != nil {
		return err
	}
	return w.Set(key, b)
}

type Digests []Digest

func (d Digests) toRequest() BatchRequest {
	bd := BatchRequest{Operations: make([]Operation, len(d))}
	for i, d := range d {
		bd.Operations[i] = d.Operation()
	}
	return bd
}

type (
	segment = confluence.Segment[BatchRequest, BatchRequest]
	source  = confluence.Source[BatchRequest]
	sink    = confluence.Sink[BatchRequest]
)

func (d Digest) Operation() Operation {
	return Operation{
		Key:         d.Key,
		Version:     d.Version,
		Leaseholder: d.Leaseholder,
		Variant:     d.Variant,
	}
}

func digestKey(key []byte) (opKey []byte, err error) {
	return kvx.CompositeKey("--dig/", key)
}
