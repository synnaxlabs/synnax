// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Key } from "@/record";

export type Primitive =
  | string
  | number
  | bigint
  | boolean
  | Stringer
  | null
  | undefined;

export interface Stringer {
  toString: () => string;
}

export const isStringer = (value: unknown): boolean =>
  value != null && typeof value === "object" && "toString" in value;

export type PrimitiveRecord = Record<Key, Primitive>;

export const primitiveIsZero = (value: Primitive): boolean => {
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
      return value === null;
  }
};
