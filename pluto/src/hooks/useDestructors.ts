import { array, type Destructor } from "@synnaxlabs/x";
import { useEffect, useMemo, useRef } from "react";

export interface UseDestructorsReturn {
  cleanup: () => void;
  set: (destructors: Destructor | Destructor[] | undefined) => void;
}

export const useDestructors = (): UseDestructorsReturn => {
  const ref = useRef<Destructor[]>([]);
  const value = useMemo(
    () => ({
      cleanup: () => {
        ref.current.forEach((destructor) => destructor());
        ref.current = [];
      },
      set: (destructors: Destructor | Destructor[] | undefined): void => {
        if (destructors == null) return;
        ref.current.push(...array.toArray(destructors));
      },
    }),
    [],
  );
  useEffect(() => value.cleanup, [value.cleanup]);
  return value;
};
