package fgrpc

import (
	"github.com/synnaxlabs/freighter"
	"go/types"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Translator[I, O freighter.Payload] interface {
	Forward(in I) (O, error)
	Backward(out O) (I, error)
}

type EmptyTranslator struct{}

var _ Translator[types.Nil, *emptypb.Empty] = EmptyTranslator{}

func (et EmptyTranslator) Forward(t types.Nil) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (et EmptyTranslator) Backward(*emptypb.Empty) (types.Nil, error) {
	return types.Nil{}, nil
}
