// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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

export type PrimitiveRecord = Record<string, Primitive>;
