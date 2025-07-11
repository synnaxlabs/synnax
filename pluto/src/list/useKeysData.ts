import { type record } from "@synnaxlabs/x";
import { useCallback } from "react";

import { type List } from "@/list";

export interface UseKeysDataReturn<K extends record.Key = record.Key>
  extends Pick<List.FrameProps<K, record.Keyed<K>>, "useListItem"> {
  data: K[];
}

export const useKeysData = <K extends record.Key = record.Key>(
  data: K[] | readonly K[],
): UseKeysDataReturn<K> => {
  const useListItem = useCallback(
    (key?: K) => {
      const option = data.find((option) => option === key);
      return option ? { key: option } : undefined;
    },
    [data],
  );
  return { data: data as K[], useListItem };
};
