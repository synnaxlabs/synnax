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
	"encoding/json/v2"
	"strconv"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// ToStructs returns the JSON form of each value as a google.protobuf.Struct. It returns
// an error when a value does not encode to a JSON object.
func ToStructs[T any](values []T) ([]*structpb.Struct, error) {
	structs := make([]*structpb.Struct, len(values))
	for i, v := range values {
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		structs[i] = &structpb.Struct{}
		if err = protojson.Unmarshal(b, structs[i]); err != nil {
			return nil, err
		}
	}
	return structs, nil
}

// FromStructs decodes the JSON form of each struct into a T, as a JSON request body
// decodes. It returns an error wrapping validate.ErrValidation, on the path of the
// struct's index, when a struct does not decode into a T.
func FromStructs[T any](structs []*structpb.Struct) ([]T, error) {
	values := make([]T, len(structs))
	for i, s := range structs {
		b, err := protojson.Marshal(s)
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(b, &values[i]); err != nil {
			return nil, validate.PathedError(
				errors.Wrap(validate.ErrValidation, err.Error()),
				strconv.Itoa(i),
			)
		}
	}
	return values, nil
}
