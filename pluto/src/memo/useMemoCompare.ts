// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Primitive } from "@synnaxlabs/x";
import { compare, deep } from "@synnaxlabs/x";
import { type DependencyList, useRef } from "react";

export const useMemoCompare = <V, D extends DependencyList>(
  factory: () => V,
  areEqual: (prevDeps: D, nextDeps: D) => boolean,
  deps: D,
): V => {
  const ref = useRef<{ deps: D; value: V }>();
  if (ref.current == null) ref.current = { deps, value: factory() };
  else if (!areEqual(ref.current.deps, deps)) ref.current = { deps, value: factory() };
  return ref.current.value;
};

export const compareArrayDeps = <T extends Primitive>(
  [a]: readonly [T[]] | [T[]],
  [b]: readonly [T[]] | [T[]],
): boolean => compare.primitiveArrays(a, b) === 0;

export const useMemoDeepEqualProps = <T extends Record<string, unknown> | undefined>(
  props: T,
): T => {
  const ref = useRef<T>();
  if (ref.current == null) ref.current = props;
  else if (!deep.equal(ref.current, props)) ref.current = props;
  return ref.current;
};

export const useMemoPrimitiveArray = <T extends Primitive>(arr: T[]): T[] => {
  const ref = useRef<T[]>();
  if (ref.current == null) ref.current = arr;
  else if (compare.primitiveArrays(ref.current, arr) !== 0) ref.current = arr;
  return ref.current;
};
