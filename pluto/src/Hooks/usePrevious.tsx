import { useEffect, useRef } from "react";

export const usePrevious = <V extends unknown>(value: V) => {
  const ref = useRef<V>(value);
  useEffect(() => {
    ref.current = value;
  }, [value]);
  return ref.current;
};
