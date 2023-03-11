// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";

import { memoize } from "proxy-memoize";
import { useSelector } from "react-redux";

import { selectWindow, selectWindowKey } from "@/selectors";
import { StoreState } from "@/state";
import { WindowState } from "@/window";

/**
 * Selects the window with the given key.
 *
 * @param key - The key of the window to select.
 * If not provided, the current window is selected.
 * @returns The window.
 */
export const useSelectWindow = (key?: string): WindowState | null =>
  useSelector(
    useCallback(
      memoize((state: StoreState) => selectWindow(state, key)),
      [key]
    )
  );

export const useSelectWindowKey = (label: string): string | null =>
  useSelector(
    useCallback(
      memoize((state: StoreState) => selectWindowKey(state, label)),
      [label]
    )
  );
