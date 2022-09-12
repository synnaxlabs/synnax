package errors

import (
	"github.com/arya-analytics/freighter/ferrors"
	"github.com/arya-analytics/x/binary"
	"github.com/samber/lo"
)

const freighterErrorType ferrors.Type = "delta.api.errors"

// FreighterType implements the ferrors.Error interface.
func (t Typed) FreighterType() ferrors.Type { return freighterErrorType }

var ecd = &binary.JSONEncoderDecoder{}

func encode(err error) string {
	tErr := err.(Typed)
	if !tErr.Occurred() {
		return string(ferrors.Nil)
	}
	return string(lo.Must(ecd.Encode(tErr)))
}

type rawError struct {
	Type Type        `json:"type" msgpack:"type"`
	Err  interface{} `json:"error" msgpack:"error"`
}

func decode(encoded string) error {
	var decoded rawError
	lo.Must0(ecd.Decode([]byte(encoded), &decoded))
	switch decoded.Type {
	case TypeValidation:
		return parseValidationError(decoded)
	default:
		return decodeMessageError(decoded)
	}
}

func init() { ferrors.Register(freighterErrorType, encode, decode) }

func decodeMessageError(raw rawError) error {
	msgMap, ok := raw.Err.(map[string]interface{})
	if !ok {
		panic("[freighter] - invalid error message")
	}
	msg := msgMap["message"].(string)
	return newTypedMessage(raw.Type, msg)
}

func parseValidationError(raw rawError) error {
	rawFields, ok := raw.Err.([]interface{})
	if !ok {
		panic("[freighter] - invalid error message")
	}
	var fields Fields
	for _, rawField := range rawFields {
		fieldMap, ok := rawField.(map[string]interface{})
		if !ok {
			panic("[freighter] - invalid error message")
		}
		fields = append(fields, Field{
			Field:   fieldMap["field"].(string),
			Message: fieldMap["message"].(string),
		})
	}
	return Validation(fields)
}
