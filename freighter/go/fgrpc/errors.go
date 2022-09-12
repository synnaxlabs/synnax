package fgrpc

import (
	"github.com/synnaxlabs/freighter/ferrors"
	v1 "github.com/synnaxlabs/freighter/ferrors/v1"
)

func EncodeError(err error) *v1.ErrorPayload {
	pld := ferrors.Encode(err)
	return &v1.ErrorPayload{Type: string(pld.Type), Data: pld.Data}
}

func DecodeError(pld *v1.ErrorPayload) error {
	return ferrors.Decode(ferrors.Payload{Type: ferrors.Type(pld.Type), Data: pld.Data})
}
