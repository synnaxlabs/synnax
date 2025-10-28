// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:generate go run gen.go

// Package op provides high-performance vectorized operations for telem.Series
// using static code generation. All operations are specialized for specific
// data types to maximize performance by eliminating type checks and virtual
// dispatch.
package op

import "github.com/synnaxlabs/x/telem"

type Binary = func(a, b telem.Series, output *telem.Series)

type Unary = func(input telem.Series, output *telem.Series)

type Reduction = func(input telem.Series, prevCount int64, output *telem.Series) int64
