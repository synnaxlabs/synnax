import { DependencyList, useRef } from "react";

export const useMemoCompare = <V, D extends DependencyList>(
  factory: () => V,
  areEqual: (prev: D, next: D) => boolean,
  deps: D
): V => {
  const ref = useRef<{ deps: D; value: V }>();
  if (ref.current == null) ref.current = { deps, value: factory() };
  else if (!areEqual(ref.current.deps, deps)) ref.current = { deps, value: factory() };
  return ref.current.value;
};
