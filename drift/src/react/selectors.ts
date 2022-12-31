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

import { StoreState } from "../state";

import { Window, WindowState } from "@/window";

/**
 * Selects the status of the window with the given key.
 *
 * @param key - The key of the window to select the status of.
 * If not provided, the status of the current window is selected.
 * @returns The status of the window.
 */
export const useSelectWindowState = (key?: string): WindowState =>
  useSelector(
    useCallback(
      memoize((state: StoreState) => {
        return state.drift.windows[key ?? state.drift.key].state;
      }),
      [key]
    )
  );

/**
 * Selects the window with the given key.
 *
 * @param key - The key of the window to select.
 * If not provided, the current window is selected.
 * @returns The window.
 */
export const useSelectWindow = (key?: string): Window =>
  useSelector(
    useCallback(
      memoize((state: StoreState) => state.drift.windows[key ?? state.drift.key]),
      [key]
    )
  );
