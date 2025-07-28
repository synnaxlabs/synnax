import { type record } from "@synnaxlabs/x";
import { useCallback } from "react";

import { type FrameProps, type GetItem } from "@/list/Frame";

export interface UseKeysDataReturn<K extends record.Key = record.Key>
  extends Required<Pick<FrameProps<K, record.Keyed<K>>, "getItem">> {
  data: K[];
}

export const useKeysData = <K extends record.Key = record.Key>(
  data: K[] | readonly K[],
): UseKeysDataReturn<K> => {
  const getItem = useCallback(
    ((key: K | K[]) => {
      if (Array.isArray(key))
        return data.filter((d) => key.includes(d)).map((d) => ({ key: d }));
      const option = data.find((option) => option === key);
      return option ? { key: option } : undefined;
    }) as GetItem<K, record.Keyed<K>>,
    [data],
  );
  return { data: data as K[], getItem };
};
