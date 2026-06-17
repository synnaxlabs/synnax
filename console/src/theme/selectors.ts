// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Theming } from "@synnaxlabs/pluto";

import { useMemoSelect } from "@/hooks";
import { SLICE_NAME, type StoreState } from "@/theme/slice";

export const selectActiveKey = (state: StoreState): string =>
  state[SLICE_NAME].active;

export const selectTheme = (
  state: StoreState,
  key?: string,
): Theming.Theme | null => {
  const { themes, active } = state[SLICE_NAME];
  const spec = themes[key ?? active];
  return spec == null ? null : Theming.themeZ.parse(spec);
};

export const useSelectTheme = (key?: string): Theming.Theme | null =>
  useMemoSelect((state: StoreState) => selectTheme(state, key), [key]);
