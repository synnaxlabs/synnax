// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { memoize } from "proxy-memoize";
import { useCallback } from "react";
import { useSelector } from "react-redux";

import {
  selectWindow,
  selectWindowAttribute,
  selectWindowKey,
  selectWindows,
} from "@/selectors";
import { type StoreState } from "@/state";
import { type WindowState } from "@/window";

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
      [key],
    ),
  );

export const useSelectWindows = (): WindowState[] =>
  useSelector(useCallback(memoize(selectWindows), []));

export const useSelectWindowKey = (label?: string): string | null =>
  useSelector(
    useCallback(
      memoize((state: StoreState) => selectWindowKey(state, label)),
      [label],
    ),
  );

export const useSelectWindowAttribute = <K extends keyof WindowState>(
  keyOrLabel: string,
  attr: K,
): WindowState[K] | null =>
  useSelector(
    useCallback(
      memoize((state: StoreState) => selectWindowAttribute(state, keyOrLabel, attr)),
      [keyOrLabel, attr],
    ),
  );
