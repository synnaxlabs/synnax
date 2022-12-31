// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
