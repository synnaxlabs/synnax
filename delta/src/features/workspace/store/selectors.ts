import { useCallback } from "react";

import memoize from "proxy-memoize";
import { useSelector } from "react-redux";

import { WorkspaceStoreState } from "./slice";
import type { Range } from "./types";

export const useSelectRangeFilterCore = (
  state: WorkspaceStoreState,
  keys?: string[]
): Range[] =>
  Object.values(state.workspace.ranges).filter(
    (range) => keys == null || keys.includes(range.key)
  );

export const useSelectSelectedRangeCore = (
  state: WorkspaceStoreState
): Range | null => {
  const { selectedRangeKey, ranges } = state.workspace;
  return selectedRangeKey != null ? ranges[selectedRangeKey] : null;
};

export const useSelectSelectedRange = (): Range | null =>
  useSelector(
    useCallback((state: WorkspaceStoreState) => useSelectSelectedRangeCore(state), [])
  );

export const useSelectRanges = (): Range[] =>
  useSelector(
    useCallback(
      memoize((state: WorkspaceStoreState) => useSelectRangeFilterCore(state)),
      []
    )
  );
