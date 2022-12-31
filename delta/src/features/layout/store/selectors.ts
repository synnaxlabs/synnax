// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { MosaicLeaf, Theme } from "@synnaxlabs/pluto";

import { Layout } from "../types";

import { LayoutStoreState } from "./slice";

import { useMemoSelect } from "@/hooks";

/**
 * Selects a layout from the store by key.
 *
 * @param state - The store state.
 * @param key - The layout key.
 * @returns The layout. Undefined if not found.
 */
export const selectLayout = (
  state: LayoutStoreState,
  key: string
): Layout | undefined => state.layout.layouts[key];

/**
 * Selects a layout from the store by key.
 *
 * @param key - The layout key.
 * @returns The layout. Undefined if not found.
 */
export const useSelectLayout = (key: string): Layout | undefined =>
  useMemoSelect((state: LayoutStoreState) => selectLayout(state, key), [key]);

/**
 * Selects the central layout mosaic from the store.
 *
 * @param state - The store state.
 * @returns The central layout mosaic.
 */
export const selectMosaic = (state: LayoutStoreState): MosaicLeaf =>
  state.layout.mosaic;

/**
 * Selects the central layout mosaic from the store.
 * @returns The central layout mosaic.
 */
export const useSelectMosaic = (): MosaicLeaf => useMemoSelect(selectMosaic);

/**
 * Selects the current theme from the store.
 *
 * @param state - The store state.
 * @returns  The current theme.
 */
export const selectTheme = (state: LayoutStoreState): Theme =>
  state.layout.themes[state.layout.activeTheme];

/**
 * Selects the current theme from the store.
 * @returns  The current theme.
 */
export const useSelectTheme = (): Theme => useMemoSelect(selectTheme);
