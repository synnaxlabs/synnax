import { useCallback } from "react";

import memoize from "proxy-memoize";
import { useSelector } from "react-redux";

import { WorkspaceStoreState } from "./slice";
import type { Range } from "./types";

export const useSelectRangeFilterCore = (
  state: WorkspaceStoreState,
  keys: string[]
): Range[] => state.workspace.ranges.filter((range) => keys.includes(range.key));

export const useSelectRanges = (): Range[] => {
  return useSelector(
    useCallback(
      memoize((state: WorkspaceStoreState) => state.workspace.ranges),
      []
    )
  );
};
