import { DependencyList, useRef } from "react";

import { comparePrimitiveArrays } from "@synnaxlabs/x";
import type { Primitive } from "@synnaxlabs/x";

export const useMemoCompare = <V, D extends DependencyList>(
  factory: () => V,
  areEqual: (prevDevps: D, nextDeps: D) => boolean,
  deps: D
): V => {
  const ref = useRef<{ deps: D; value: V }>();
  if (ref.current == null) ref.current = { deps, value: factory() };
  else if (!areEqual(ref.current.deps, deps)) ref.current = { deps, value: factory() };
  return ref.current.value;
};

export const compareArrayDeps = <T extends Primitive>(
  [a]: readonly [T[]],
  [b]: readonly [T[]]
): boolean => comparePrimitiveArrays(a, b) === 0;
