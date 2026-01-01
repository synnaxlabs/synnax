// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package errors

import (
	"context"
	"strings"
)

// Payload is a typed payload for transporting an error OVER a NETWORK.
// It includes type information as well as encoded error data.
type Payload struct {
	// Type is the type of the error.
	Type string `json:"type" msgpack:"type"`
	// Data is the encoded error data.
	Data string `json:"data" msgpack:"data"`
}

// Error implements the error interface.
func (p Payload) Error() string {
	return string(p.Type) + "---" + p.Data
}

func (p *Payload) Unmarshal(d string) {
	a := strings.Split(d, "---")
	if len(a) != 2 {
		p.Type = TypeUnknown
		p.Data = d
	} else {
		p.Type = a[0]
		p.Data = a[1]
	}
}

var _registry = newRegistry()

// Register registers an error type with the given type.
func Register(encode EncodeFunc, decode DecodeFunc) {
	_registry.register(provider{encode: encode, decode: decode})
}

// Encode encodes an error into a payload. If the type of the error cannot be
// determined, returns a payload with type TypeUnknown and the error message. If
// the error is nil, returns a payload with type TypeNil.
func Encode(ctx context.Context, e error, internal bool) Payload {
	pld, ok := e.(Payload)
	if ok {
		return pld
	}
	return _registry.encode(ctx, e, internal)
}

// Decode decodes a payload into an error. If the payload's type is TypeUnknown,
// returns an error with the payload's data as the message. If the payload's
// type is TypeNil, returns nil.
func Decode(ctx context.Context, p Payload) error { return _registry.decode(ctx, p) }

type EncodeFunc func(context.Context, error) (Payload, bool)

type DecodeFunc func(context.Context, Payload) (error, bool)

type provider struct {
	encode EncodeFunc
	decode DecodeFunc
}

// registry is a registry of error providers. It is used to encode and decode errors
// into payloads for transport over the network.
type registry struct {
	providers []provider
}

func newRegistry() *registry {
	return &registry{providers: make([]provider, 0)}
}

func (r *registry) register(e provider) {
	r.providers = append(r.providers, e)
}

func (r *registry) encode(ctx context.Context, e error, internal bool) Payload {
	// If the error is nil, return a standardized payload.
	if e == nil {
		return Payload{Type: TypeNil}
	}
	for _, p := range r.providers {
		if payload, ok := p.encode(ctx, e); ok {
			return payload
		}
	}
	if internal {
		return roachEncode(ctx, e)
	}
	return Payload{Type: TypeUnknown, Data: e.Error()}
}

func (r *registry) decode(ctx context.Context, p Payload) error {
	if p.Type == TypeNil || (p.Type == "" && p.Data == "") {
		return nil
	}
	for _, prov := range r.providers {
		if err, ok := prov.decode(ctx, p); ok {
			return err
		}
	}
	if err, ok := roachDecode(ctx, p); ok {
		return err
	}
	return New(p.Data)
}
