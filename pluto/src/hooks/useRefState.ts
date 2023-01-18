import { MutableRefObject, RefObject, useCallback, useRef } from "react";

export type PseudoSetState<T> = (value: T | ((prev: T) => T)) => void;

export const useRefState = <T extends object>(
  initialValue: T
): [RefObject<T> | MutableRefObject<T>, PseudoSetState<T>] => {
  const ref = useRef<T>(initialValue);
  const setValue: PseudoSetState<T> = useCallback((value) => {
    if (typeof value === "function") ref.current = value(ref.current);
    else ref.current = value;
  }, []);
  return [ref, setValue];
};
