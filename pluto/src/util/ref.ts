import { Ref } from "react";

export const triggerRef = <T>(ref: Ref<T> | undefined, v: T): void => {
  if (typeof ref === "function") ref(v);
  // @ts-expect-error
  else if (ref != null) ref.current = v;
};

export const mergeRefs =
  <T>(...refs: Array<Ref<T> | undefined>) =>
  (e: T | null) => {
    refs.forEach((ref) => triggerRef(ref, e));
  };
