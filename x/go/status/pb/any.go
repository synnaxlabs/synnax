// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb

import (
	"context"
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

// AnyFromPBAny converts *anypb.Any to any for use with Status[any].
// It handles both JSON objects (structpb.Struct) and typed protos.
// For typed protos, it uses protojson to convert to a JSON-compatible map.
func AnyFromPBAny(_ context.Context, a *anypb.Any) (any, error) {
	if a == nil {
		return nil, nil
	}

	// Try structpb.Struct first (JSON object case)
	s := &structpb.Struct{}
	if err := a.UnmarshalTo(s); err == nil {
		return s.AsMap(), nil
	}

	// For typed protos, unmarshal using the type registry and convert to JSON
	msg, err := anypb.UnmarshalNew(a, proto.UnmarshalOptions{})
	if err != nil {
		// If we can't unmarshal, return nil (unknown type)
		return nil, nil
	}

	// Convert the proto message to JSON
	jsonBytes, err := protojson.Marshal(msg)
	if err != nil {
		return nil, err
	}

	// Parse JSON into a map
	var result map[string]any
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// AnyToPBAny converts any to *anypb.Any for use with Status[any].
// It wraps values in structpb.Struct since status details are always objects.
func AnyToPBAny(_ context.Context, v any) (*anypb.Any, error) {
	if v == nil {
		return nil, nil
	}

	// If it's already *anypb.Any, return it directly
	if a, ok := v.(*anypb.Any); ok {
		return a, nil
	}

	// If it's a proto.Message, pack it directly
	if pm, ok := v.(proto.Message); ok {
		return anypb.New(pm)
	}

	// Convert to map for structpb.NewStruct
	m, ok := v.(map[string]any)
	if !ok {
		// Try to convert via JSON marshaling
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(jsonBytes, &m); err != nil {
			return nil, err
		}
	}

	// Wrap in structpb.Struct (for JSON objects)
	s, err := structpb.NewStruct(m)
	if err != nil {
		return nil, err
	}
	return anypb.New(s)
}