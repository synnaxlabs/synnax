// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** Union of types that are primitive values or can be converted to primitive values */
export type Value = string | number | bigint | boolean | Stringer | null | undefined;

export type CrudeValueExtension<V extends NonNullable<Value>> = {
  value: V;
};

export const isCrudeValueExtension = <V extends NonNullable<Value>>(
  value: unknown,
): value is CrudeValueExtension<V> =>
  value != null && typeof value === "object" && "value" in value;

/**
 * ValueExtension is a utility class that can be extended in order to implement objects
 * that pseudo-extend a primitive value with additional functionality.
 */
export class ValueExtension<V extends NonNullable<Value>> {
  /** The underlying primitive value */
  protected readonly value: V;

  constructor(value: V) {
    this.value = value;
  }

  /** Overrides the JS default valueOf() function to return the primitive value. */
  valueOf(): V {
    return this.value;
  }

  /** toJSON ensures that only the primitive value gets encoded during JSON
   * stringification. */
  toJSON(): V {
    return this.value;
  }

  /** @returns a string representation of the item. */
  toString(): string {
    return this.value.toString();
  }
}

/**
 * Stringer is a type that implements a toString() method in order to return a string
 * representation of itself.
 */
export interface Stringer {
  /** @returns a string representation of the item. */
  toString: () => string;
}

/** @returns true if the value implements primitive.Stringer, otherwise returns false. */
export const isStringer = (value: unknown): boolean =>
  value != null && typeof value === "object" && "toString" in value;

/**
 * Type representing zero values for each primitive type
 */
export type ZeroValue = "" | 0 | 0n | false | null | undefined;

/**
 * Type representing non-zero values for each primitive type
 */
export type NonZeroValue = Exclude<Value, ZeroValue>;

/**
 * @returns true if the given primitive is the zero value for its type.
 * For strings value == ""
 * For numbers value == 0
 * For bigints value == 0n
 * For booleans value == false
 * For objects value == null
 * For undefined returns true
 */
export const isZero = <V extends Value>(value: V): value is V & ZeroValue => {
  if (isStringer(value)) return value?.toString().length === 0;
  switch (typeof value) {
    case "string":
      return value.length === 0;
    case "number":
      return value === 0;
    case "bigint":
      return value === 0n;
    case "boolean":
      return !value;
    case "undefined":
      return true;
    case "object":
      return value == null;
    default:
      return false;
  }
};

/**
 * Type predicate function that narrows to non-zero values
 */
export const isNonZero = <V extends Value>(value: V): value is V & NonZeroValue =>
  !isZero(value);
