// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Freezes the value and every object it holds, in place.
 * @param value - The value to freeze. Primitives are returned unchanged.
 * @returns The same value, now deeply frozen.
 */
export const freeze = <T>(value: T): T => {
  if (typeof value !== "object" || value === null || Object.isFrozen(value))
    return value;
  Object.freeze(value);
  Object.values(value).forEach(freeze);
  return value;
};
