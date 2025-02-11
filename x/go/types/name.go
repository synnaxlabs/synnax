// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import "reflect"

type CustomTypeName interface {
	CustomTypeName() string
}

func Name[T any]() string {
	var t T
	if ct, ok := any(t).(CustomTypeName); ok {
		return ct.CustomTypeName()
	}
	return reflect.TypeOf(*new(T)).Name()
}

func PluralName[T any]() string {
	name := Name[T]()
	return name + "s"
}
