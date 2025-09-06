// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export type IsExactlyUndefined<T> = [T] extends [undefined] // T can be assigned to undefined
  ? [undefined] extends [T] // undefined can be assigned to T
    ? true // both directions → exactly undefined
    : false
  : false;
