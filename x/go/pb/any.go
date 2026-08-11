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
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

// AnyFromPBAny converts *anypb.Any to any. It handles JSON values of every type
// (structpb.Value), JSON objects sent by peers on releases before value packing
// (structpb.Struct), and typed protos. For typed protos, it uses protojson to convert
// to a JSON-compatible map.
func AnyFromPBAny(a *anypb.Any) (any, error) {
	if a == nil {
		return nil, nil
	}

	v := &structpb.Value{}
	if err := a.UnmarshalTo(v); err == nil {
		return v.AsInterface(), nil
	}

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

// AnyToPBAny converts any to *anypb.Any. It wraps JSON values of every type in a
// structpb.Value, passes through proto.Message and *anypb.Any directly, and falls back
// to JSON marshaling for types structpb does not accept directly.
func AnyToPBAny(v any) (*anypb.Any, error) {
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

	val, err := structpb.NewValue(v)
	if err != nil {
		jsonBytes, marshalErr := json.Marshal(v)
		if marshalErr != nil {
			return nil, marshalErr
		}
		var decoded any
		if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
			return nil, err
		}
		if val, err = structpb.NewValue(decoded); err != nil {
			return nil, err
		}
	}
	return anypb.New(val)
}
