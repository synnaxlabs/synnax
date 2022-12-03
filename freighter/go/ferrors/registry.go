package ferrors

import (
	"context"
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
)

// Payload is a typed payload for transporting an error OVER a NETWORK.
// It includes type information as well as encoded error data.
type Payload struct {
	// Type is the type of the error.
	Type Type `json:"type" msgpack:"type"`
	// Data is the encoded error data.
	Data string `json:"data" msgpack:"data"`
}

var _registry = newRegistry()

type EncodeFunc func(error) string

type DecodeFunc func(string) error

// Register registers an error type with the given type.
func Register(t Type, encode EncodeFunc, decode DecodeFunc) {
	_registry.register(t, provider{encode: encode, decode: decode})
}

// Encode encodes an error into a payload. If the type of the error cannot be
// determined, returns a payload with type Unknown and the error message. If
// the error is nil, returns a payload with type Nil.
func Encode(e error) Payload { return _registry.encode(e) }

// Decode decodes a payload into an error. If the payload's type is Unknown,
// returns an error with the payload's data as the message. If the payload's
// type is Nil, returns nil.
func Decode(p Payload) error { return _registry.decode(p) }

type provider struct {
	encode EncodeFunc
	decode DecodeFunc
}

// registry is a registry of error providers. It is used to encode and decode errors
// into payloads for transport over the network.
type registry struct {
	providers map[Type]provider
}

func newRegistry() *registry {
	return &registry{providers: make(map[Type]provider)}
}

func (r *registry) register(t Type, e provider) {
	if _, ok := r.providers[t]; ok {
		panic("[freighter.errors.Errors] - type already registered")
	}
	r.providers[t] = e
}

func (r *registry) encode(e error) Payload {

	// If the error is nil, return a standardized payload.
	if e == nil {
		return Payload{Type: Nil}
	}

	tErr, ok := e.(Error)
	if !ok {

		// If the type isn't registered, attempt to encode the error using
		// cockroachdb's error package. This used for go-to-go transport.
		encoded := errors.EncodeError(context.TODO(), e)
		b, err := encoded.Marshal()

		// If we couldn't encode the error, return a standardized unknown
		// payload along with the error string.
		if err != nil {
			return Payload{Type: Unknown, Data: err.Error()}
		}

		return Payload{Type: Roach, Data: string(b)}
	}

	prov, ok := r.providers[tErr.FreighterType()]
	if !ok {
		zap.L().Sugar().Warnf(
			"[freighter.errors.Errors] - type %s not registered. returning unknown payload",
			tErr.FreighterType(),
		)
		return Payload{Type: Unknown, Data: e.Error()}
	}

	data := prov.encode(e)

	// If the caller has a nil representation of an error type, we expect them
	// to return a Nil payload so we can convert it to a freighter payload.
	if data == string(Nil) {
		return Payload{Type: Nil}
	}

	return Payload{Type: tErr.FreighterType(), Data: data}
}

func (r *registry) decode(p Payload) error {
	if p.Type == Nil {
		return nil
	}
	if p.Type == Roach {
		e := &errors.EncodedError{}
		if err := e.Unmarshal([]byte(p.Data)); err != nil {
			return err
		}
		return errors.DecodeError(context.TODO(), *e)
	}
	if e, ok := r.providers[p.Type]; ok {
		return e.decode(p.Data)
	}
	return errors.New(p.Data)
}
