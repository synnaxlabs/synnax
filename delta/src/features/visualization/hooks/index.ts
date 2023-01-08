import { useCallback } from "react";

import { useDispatch } from "react-redux";

import { updateVisualization as uv } from "../store";
import { Visualization } from "../types";

export const useUpdateVisualization = (
  key: string
): (<V extends Visualization>(v: V) => void) => {
  const d = useDispatch();
  return useCallback(
    <V extends Visualization>(v: V): void => {
      d(uv({ ...v, key }));
    },
    [d, key]
  );
};
