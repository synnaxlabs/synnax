// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ferrors

import (
	"context"
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
	"strings"
)

// Payload is a typed payload for transporting an error OVER a NETWORK.
// It includes type information as well as encoded error data.
type Payload struct {
	// Type is the type of the error.
	Type Type `json:"type" msgpack:"type"`
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
		p.Type = Type(a[0])
		p.Data = a[1]
	}
}

var _registry = newRegistry()

// Register registers an error type with the given type.
func Register(encode EncodeFunc, decode DecodeFunc) {
	_registry.register(providerFunc{encode: encode, decode: decode})
}

// Encode encodes an error into a payload. If the type of the error cannot be
// determined, returns a payload with type TypeUnknown and the error message. If
// the error is nil, returns a payload with type TypeNil.
func Encode(e error) Payload { return _registry.encode(e) }

// Decode decodes a payload into an error. If the payload's type is TypeUnknown,
// returns an error with the payload's data as the message. If the payload's
// type is TypeNil, returns nil.
func Decode(p Payload) error { return _registry.decode(p) }

type EncodeFunc func(context.Context, error) (Payload, bool)

type DecodeFunc func(context.Context, Payload) (error, bool)

type Provider interface {
	Encode(context.Context, error) (Payload, bool)
	Decode(context.Context, Payload) (error, bool)
}

type providerFunc struct {
	encode EncodeFunc
	decode DecodeFunc
}

var _ Provider = providerFunc{}

func (p providerFunc) Encode(ctx context.Context, err error) (Payload, bool) {
	return p.encode(ctx, err)
}

func (p providerFunc) Decode(ctx context.Context, pld Payload) (error, bool) {
	return p.decode(ctx, pld)
}

// registry is a registry of error providers. It is used to encode and decode errors
// into payloads for transport over the network.
type registry struct {
	providers []Provider
}

func newRegistry() *registry {
	return &registry{providers: make([]Provider, 0)}
}

func (r *registry) register(e Provider) {
	r.providers = append(r.providers, e)
}

func (r *registry) encode(e error) Payload {
	// If the error is nil, return a standardized payload.
	if e == nil {
		return Payload{Type: TypeNil}
	}

	for _, p := range r.providers {
		if payload, ok := p.Encode(context.Background(), e); ok {
			return payload
		}
	}

	var tErr Error
	if errors.As(e, &tErr) {
		zap.L().Sugar().Warnf(
			"[freighter.errors.Errors] - type %s not registered. returning unknown payload",
			tErr.FreighterType(),
		)
	}
	return Payload{Type: TypeUnknown, Data: e.Error()}
}

func (r *registry) decode(p Payload) error {
	if p.Type == TypeNil {
		return nil
	}
	for _, prov := range r.providers {
		if err, ok := prov.Decode(context.Background(), p); ok {
			return err
		}
	}
	return errors.New(p.Data)
}
