// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { compare, deep, type primitive } from "@synnaxlabs/x";
import { type DependencyList, useRef } from "react";

const arraysEqual = <T>(a: readonly T[], b: readonly T[]): boolean => {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) if (!Object.is(a[i], b[i])) return false;
  return true;
};

export const useMemoCompare = <V, D extends DependencyList>(
  factory: () => V,
  areEqual: (prevDeps: D, nextDeps: D) => boolean,
  deps: D,
): V => {
  const ref = useRef<{ deps: D; value: V }>(null);
  if (ref.current == null) ref.current = { deps, value: factory() };
  else if (!areEqual(ref.current.deps, deps)) ref.current = { deps, value: factory() };
  return ref.current.value;
};

export const compareArrayDeps = <T extends primitive.Value>(
  [a]: readonly [T[]] | [T[]],
  [b]: readonly [T[]] | [T[]],
): boolean => compare.primitiveArrays(a, b) === 0;

export const useMemoDeepEqual = <T>(value: T): T => {
  const ref = useRef<T>(null);
  if (ref.current == null) ref.current = value;
  else if (!deep.equal(ref.current, value)) ref.current = value;
  return ref.current;
};

/**
 * Returns a referentially stable array as long as its contents are element-wise equal
 * (per `Object.is`, matching React's dep comparison) to the previous render's
 * contents. Faster than {@link useMemoDeepEqual} since it short-circuits on length
 * and avoids structural traversal.
 */
export const useMemoArray = <T>(value: readonly T[]): readonly T[] => {
  const ref = useRef<readonly T[]>(null);
  if (ref.current == null || !arraysEqual(ref.current, value)) ref.current = value;
  return ref.current;
};
