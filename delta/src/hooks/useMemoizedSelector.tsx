import { useCallback } from "react";

import memoize from "proxy-memoize";
import { useSelector } from "react-redux";

export const useMemoSelect = <S extends object, R>(
  selector: (state: S) => R,
  deps: unknown[]
): R => useSelector(useCallback(memoize(selector), deps));
