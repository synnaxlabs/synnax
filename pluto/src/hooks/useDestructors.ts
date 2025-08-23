import { array, type Destructor } from "@synnaxlabs/x";
import { useCallback, useEffect, useRef } from "react";

export interface UseDestructorsReturn {
  cleanup: () => void;
  set: (destructors: Destructor | Destructor[] | undefined) => void;
}

export const useDestructors = (): UseDestructorsReturn => {
  const ref = useRef<Destructor[]>([]);
  const cleanup = useCallback((): void => {
    ref.current.forEach((destructor) => destructor());
    ref.current = [];
  }, []);
  const set = useCallback(
    (destructors: Destructor | Destructor[] | undefined): void => {
      if (destructors == null) return;
      ref.current.push(...array.toArray(destructors));
    },
    [cleanup],
  );
  useEffect(() => cleanup, [cleanup]);
  return { cleanup, set };
};
