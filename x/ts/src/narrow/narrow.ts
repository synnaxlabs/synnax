// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@/record";

export type IsUndefined<T> = [T] extends [undefined] // T can be assigned to undefined
  ? [undefined] extends [T] // undefined can be assigned to T
    ? true // both directions → exactly undefined
    : false
  : false;

export const isObject = <T extends record.Unknown = record.Unknown>(
  item?: unknown,
): item is T => item != null && typeof item === "object" && !Array.isArray(item);
