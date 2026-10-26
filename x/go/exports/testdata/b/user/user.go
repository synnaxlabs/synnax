// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package user

import "example.com/a/lib"

func Run() (lib.Shape, lib.Op, lib.Color) {
	var _ lib.DoExtra = lib.Outer{}
	_ = lib.Big
	lib.Used()
	lib.Open(lib.Config{})
	return lib.Square{}, lib.Op{}, lib.Red
}
