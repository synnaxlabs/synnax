// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { compare, deep, type primitive } from "@synnaxlabs/x";
import { type DependencyList, useState } from "react";

/** @returns whether two lists hold the same elements, as React compares deps. */
export const compareDeps = <T>(a: readonly T[], b: readonly T[]): boolean => {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) if (!Object.is(a[i], b[i])) return false;
  return true;
};

/**
 * Keeps one value until the comparison rejects a change, then stores the next one.
 * Storing it during render re-runs the caller's render at once, before any commit.
 */
const useStable = <T>(value: T, areEqual: (prev: T, next: T) => boolean): T => {
  const [stored, setStored] = useState<T>(() => value);
  if (Object.is(stored, value) || areEqual(stored, value)) return stored;
  setStored(() => value);
  return value;
};

/**
 * Like `useMemo`, but recomputes only when `areEqual` rejects the new deps. The factory
 * runs once per accepted change.
 */
export const useMemoCompare = <V, D extends DependencyList>(
  factory: () => V,
  areEqual: (prevDeps: D, nextDeps: D) => boolean,
  deps: D,
): V => {
  const [memo, setMemo] = useState(() => ({ deps, value: factory() }));
  if (areEqual(memo.deps, deps)) return memo.value;
  const next = { deps, value: factory() };
  setMemo(next);
  return next.value;
};

export const compareArrayDeps = <T extends primitive.Value>(
  [a]: readonly [T[]] | [T[]],
  [b]: readonly [T[]] | [T[]],
): boolean => compare.primitiveArrays(a, b) === 0;

/**
 * @returns the first value it was given for as long as each new value is deeply equal
 * to it, so the result can serve as an effect or memo dependency.
 */
export const useMemoDeepEqual = <T>(value: T): T => useStable(value, deep.equal);

/**
 * Returns a referentially stable array as long as its contents are element-wise equal
 * (per `Object.is`, matching React's dep comparison) to the previous render's
 * contents. Faster than {@link useMemoDeepEqual} since it short-circuits on length
 * and avoids structural traversal.
 */
export const useMemoArray = <T>(value: readonly T[]): readonly T[] =>
  useStable(value, compareDeps);
