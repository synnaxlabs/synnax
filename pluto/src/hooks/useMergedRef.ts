import { MutableRefObject, Ref, RefCallback, useCallback } from "react";

export const useMergedRef = <T>(...refs: Array<Ref<T>>): RefCallback<T> => {
  return useCallback(
    (el: T) => {
      refs.forEach((ref) => {
        if (typeof ref === "function") ref(el);
        else if (ref != null) (ref as MutableRefObject<T>).current = el;
      });
    },
    [refs]
  );
};
