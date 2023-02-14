// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Primitive, isStringer, PrimitiveRecord } from "@/types/primitive";

export type CompareF<T> = (a: T, b: T) => number;


export type PrimitiveCompareF<T extends Primitive> = CompareF<T>;

export const primitiveCompareFactory = <T extends Primitive>(
  v: T,
  reverse: boolean = false
): PrimitiveCompareF<T> => {
  const t = isStringer(v) ? "stringer" : typeof v;
  let f: CompareF<T>;
  switch (t) {
    case "string" || "stringer":
      f = (a: T, b: T) =>
        (a as string).toString().localeCompare((b as string).toString());
      break;
    case "number" || "bigint":
      f = (a: T, b: T) => (a as number) - (b as number);
      break;
    case "boolean":
      f = (a: T, b: T) => Number(a) - Number(b);
      break;
    default:
      console.warn("sortFunc: unknown type");
      return () => -1;
  }
  return reverse ? reverseCompare(f) : f;
};

export const comparePrimitiveArrays = <T extends Primitive>(
  a: readonly T[] | T[],
  b: readonly T[] | T[]
): number => {
  if (a.length !== b.length) return a.length - b.length;
  return a.every((v, i) => v === b[i]) ? 0 : -1;
};

export const reverseCompare =
  <T>(f: CompareF<T>): CompareF<T> =>
  (a: T, b: T) =>
    f(b, a);

export const objectValueCompareFactory = <T extends PrimitiveRecord>(
  key: keyof T,
  value: T,
  reverse: boolean = false
): CompareF<T> => {
  const f = primitiveCompareFactory(value[key], reverse);
  return (a: T, b: T) => f(a[key], b[key]);
};
