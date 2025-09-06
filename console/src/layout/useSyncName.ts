import { usePrevious } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { useEffect } from "react";
import { useDispatch } from "react-redux";

import { useSelect } from "@/layout/selectors";
import { rename } from "@/layout/slice";

export const useSyncName = (
  layoutKey: string,
  externalName: string,
  onChange: (name: string) => void,
) => {
  const layoutName = useSelect(layoutKey)?.name;
  const prevLayoutName = usePrevious(layoutName);
  const dispatch = useDispatch();
  useEffect(() => {
    if (prevLayoutName == layoutName || prevLayoutName == null || layoutName == null)
      return;
    onChange(layoutName);
  }, [layoutName, onChange, prevLayoutName]);
  useEffect(() => {
    if (primitive.isNonZero(externalName))
      dispatch(rename({ key: layoutKey, name: externalName }));
  }, [externalName]);
};
