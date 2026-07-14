// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Theming } from "@synnaxlabs/pluto";
import { useMemo } from "react";

import { Select } from "@/session/select";
import {
  type Mode,
  modeZ,
  SLICE_NAME,
  type SliceState,
  type StoreState,
} from "@/session/theme/slice";
import { useOSTheme } from "@/session/theme/useOSTheme";

const selectSlice = (state: StoreState): SliceState => state[SLICE_NAME];

const selectMode = (state: StoreState): Mode =>
  modeZ.catch("system").parse(selectSlice(state).mode);

export const useSelectMode = (): Mode => Select.useMemo(selectMode, []);

const FIXED_KEYS: Record<Exclude<Mode, "system">, string> = {
  light: "synnaxLight",
  dark: "synnaxDark",
};

export const useProviderProps = (): Theming.ProviderProps => {
  const mode = useSelectMode();
  const osKey = useOSTheme(mode === "system");
  const key = mode === "system" ? osKey : FIXED_KEYS[mode];
  return useMemo<Theming.ProviderProps>(() => ({ theme: { key } }), [key]);
};
