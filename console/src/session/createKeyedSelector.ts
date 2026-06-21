import { type record } from "@synnaxlabs/x";

import { useMemoSelect } from "@/hooks";

export const createKeyedSelectorFactory =
  <K extends record.Key>(useKey: (override?: K) => K) =>
  <S extends object, R>(selector: (state: S, key: K) => R): ((override?: K) => R) =>
  (override) => {
    const key = useKey(override);
    return useMemoSelect((state: S) => selector(state, key), [key]);
  };
