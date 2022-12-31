// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package counter

type Uint16 interface {
	Add(delta ...uint16) uint16
	Value() uint16
}

type Uint16Error interface {
	Uint16
	Error
}

type Int32 interface {
	Add(delta ...int32) int32
	Value() int32
}

type Int64 interface {
	Add(delta ...int64) int64
	Value() int64
}

type Error interface {
	Error() error
}
