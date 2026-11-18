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
	"reflect"
	"strings"
)

// Payload is a typed payload for transporting an error OVER a NETWORK. It includes type
// information as well as encoded error data.
type Payload struct {
	// Type is the type of the error.
	Type string `json:"type" msgpack:"type"`
	// Data is the encoded error data.
	Data string `json:"data" msgpack:"data"`
}

var payloadType = reflect.TypeFor[Payload]()

// Error implements the error interface.
func (p Payload) Error() string { return p.Type + "---" + p.Data }

// ParsePayload parses the text of Payload.Error. Text in any other form becomes the
// data of a payload of type TypeUnknown.
func ParsePayload(d string) Payload {
	a := strings.Split(d, "---")
	if len(a) != 2 {
		return Payload{Type: TypeUnknown, Data: d}
	}
	return Payload{Type: a[0], Data: a[1]}
}

// EncodeFunc encodes err into a payload, or returns false if it does not handle err.
// The codec lets a provider encode the errors that err wraps.
type EncodeFunc func(c *Codec, err error) (Payload, bool)

// DecodeFunc decodes p into an error, or returns false if it does not handle p. The
// codec lets a provider decode the errors that p wraps.
type DecodeFunc func(c *Codec, p Payload) (error, bool)

// Provider encodes and decodes the errors of one package.
type Provider struct {
	Encode EncodeFunc
	Decode DecodeFunc
}

// Codec encodes errors into payloads for transport over a network and decodes them
// back. It tries its providers in order. A Codec is safe for concurrent use.
type Codec struct{ providers []Provider }

// NewCodec returns a Codec that tries the given providers in order.
func NewCodec(providers ...Provider) *Codec { return &Codec{providers: providers} }

// Encode encodes an error into a payload. If no provider handles the error, Encode
// returns a payload with type TypeRoach when internal is true, and a payload with type
// TypeUnknown and the error message when it is false. If the error is nil, returns a
// payload with type TypeNil. A bare Payload passes through unchanged.
func (c *Codec) Encode(err error, internal bool) Payload {
	// Only a bare payload passes through, so a wrapped one keeps its wrap message.
	var bare Payload
	if reflect.TypeOf(err) == payloadType && As(err, &bare) {
		return bare
	}
	if err == nil {
		return Payload{Type: TypeNil}
	}
	if pld := c.encode(err); pld != nil {
		return *pld
	}
	if internal {
		return roachEncode(err)
	}
	return Payload{Type: TypeUnknown, Data: err.Error()}
}

// Expected returns true if a provider encodes err. Such errors, for example
// validation, not found, and unauthorized errors, report a fault in a request, not in
// the code, so their stack traces have no use. A joined error is expected only if
// every member is. Expected returns false for nil.
func (c *Codec) Expected(err error) bool {
	if err == nil {
		return false
	}
	if members := UnwrapMulti(err); members != nil {
		for _, m := range members {
			if !c.Expected(m) {
				return false
			}
		}
		return true
	}
	return c.encode(err) != nil
}

func (c *Codec) encode(err error) *Payload {
	for _, p := range c.providers {
		if pld, ok := p.Encode(c, err); ok {
			return &pld
		}
	}
	return nil
}

// Decode decodes a payload into an error. If the payload's type is TypeNil, or the
// payload is empty, returns nil. If no provider handles the payload, returns an error
// with the payload's data as the message.
func (c *Codec) Decode(p Payload) error {
	if p.Type == TypeNil || (p.Type == "" && p.Data == "") {
		return nil
	}
	for _, prov := range c.providers {
		if err, ok := prov.Decode(c, p); ok {
			return err
		}
	}
	if err, ok := roachDecode(p); ok {
		return err
	}
	return New(p.Data)
}
