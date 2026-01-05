// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const isDefined = (key: symbol): boolean =>
  Object.getOwnPropertySymbols(globalThis).includes(key);

/**
 * Defines a new global singleton instance of a value.
 *
 * @param key - The unique identifier for the singleton.
 * @param value - A function that returns the singleton instance.
 * @returns A function that returns the singleton instance.
 */
export const define = <T>(key: string, value: () => T): (() => T) => {
  const symbol = Symbol.for(key);
  if (!isDefined(symbol)) {
    const singleton = value();
    Object.defineProperty(globalThis, symbol, { value: singleton });
  }
  return () => (globalThis as any)[symbol] as T;
};
