// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

/**
 * Zod schema for validating record keys. Can be either a string or number.
 */
export const keyZ = z.union([z.string(), z.number()]);

/**
 * Represents valid key types for record objects. Can be either a string or number.
 */
export type Key = z.infer<typeof keyZ>;

/**
 * Zod schema for validating unknown records. Accepts objects with string or number keys
 * and unknown values.
 */
export const unknownZ = z.record(keyZ, z.unknown());

/**
 * Represents a record with unknown values and string/number keys.
 * This is a generic type for objects where the value types are not known.
 */
export interface Unknown extends z.infer<typeof unknownZ> {}

/**
 * Interface for objects that have a key property.
 * @template K - The type of the key (must extend Key)
 */
export interface Keyed<K extends Key = Key> {
  /** The key identifier for this object */
  key: K;
}

/**
 * Interface for objects that have both a key and a name property.
 * @template K - The type of the key (defaults to string)
 */
export interface KeyedNamed<K extends Key = string> {
  /** The key identifier for this object */
  key: K;
  /** The display name for this object */
  name: string;
}

/**
 * Type representing the entries of a record as an array of key-value tuples.
 * @template T - The record type
 */
export type Entries<T> = Array<
  {
    [K in keyof T]: [K, T[K]];
  }[keyof T]
>;

/**
 * Converts a record object into an array of key-value tuples.
 * This is a type-safe wrapper around Object.entries().
 *
 * @template T - The type of the input record
 * @param obj - The record object to convert to entries
 * @returns An array of [key, value] tuples
 *
 * @example
 * ```typescript
 * const obj = { a: 1, b: "hello" };
 * const entries = record.entries(obj);
 * // Result: [["a", 1], ["b", "hello"]]
 * ```
 */
export const entries = <T extends Record<Key, unknown>>(obj: T): Entries<T> =>
  Object.entries(obj) as Entries<T>;

/**
 * Maps over the entries of a record, applying a transformation function to each value.
 *
 * @template T - The type of the input record
 * @template U - The type of the output values
 * @param obj - The record object to map over
 * @param fn - A function that transforms each value. Receives the value and key as parameters.
 * @returns A new record with the same keys but transformed values
 *
 * @example
 * ```typescript
 * const obj = { a: 1, b: 2, c: 3 };
 * const doubled = record.map(obj, (value, key) => value * 2);
 * // Result: { a: 2, b: 4, c: 6 }
 * ```
 */
export const map = <T extends Record<Key, unknown>, U>(
  obj: T,
  fn: (value: T[keyof T], key: keyof T) => U,
): Record<Key, U> =>
  Object.fromEntries(entries(obj).map(([key, value]) => [key, fn(value, key)]));

/**
 * Removes all properties with undefined or null values from a record.
 *
 * @template T - The type of the input record
 * @param obj - The record object to purge
 * @returns A new record with undefined/null values removed
 *
 * @example
 * ```typescript
 * const obj = { a: 1, b: undefined, c: null, d: "hello" };
 * const purged = record.purgeUndefined(obj);
 * // Result: { a: 1, d: "hello" }
 * ```
 */
export const purgeUndefined = <T extends Record<Key, unknown>>(obj: T): T =>
  Object.fromEntries(entries(obj).filter(([_, value]) => value !== undefined)) as T;

/**
 * Removes specified keys from an object. This creates a shallow copy of the object and
 * removes the keys instead of mutating the original object.
 *
 * @template T - The type of the input object
 * @template K - The type of the keys to remove
 * @param obj - The object to remove keys from
 * @param keys - The keys to remove from the object
 * @returns A new object with the specified keys removed
 *
 * @example
 * ```typescript
 * const obj = { a: 1, b: 2, c: 3 };
 * const omitted = record.omit(obj, "b", "c");
 * // Result: { a: 1 }
 * ```
 */
export const omit = <T, K extends keyof T>(obj: T, ...keys: K[]): Omit<T, K> => {
  const result = { ...obj };
  for (const key of keys) delete result[key];
  return result;
};
