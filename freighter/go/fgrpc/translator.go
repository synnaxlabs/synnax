// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fgrpc

import (
	"context"
	"github.com/synnaxlabs/freighter"
	"go/types"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Translator is an entity that can translate payloads from one type to another. It is
// mainly used to create separation between protobuf types and application internal types.
type Translator[I, O freighter.Payload] interface {
	// Forward translates the given input into a transportable output.
	Forward(ctx context.Context, in I) (O, error)
	// Backward translates the given output into an application internal input.
	Backward(ctx context.Context, out O) (I, error)
}

// EmptyTranslator is a translator for an empty GRPC request.
type EmptyTranslator struct{}

var _ Translator[types.Nil, *emptypb.Empty] = EmptyTranslator{}

// Forward implements Translator.
func (et EmptyTranslator) Forward(ctx context.Context, t types.Nil) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

// Backward implements Translator.
func (et EmptyTranslator) Backward(context.Context, *emptypb.Empty) (types.Nil, error) {
	return types.Nil{}, nil
}
