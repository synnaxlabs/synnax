// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Theming, useAsyncEffect } from "@synnaxlabs/pluto";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { useMemo, useState } from "react";

import { Runtime } from "@/session/runtime";
import { Select } from "@/session/select";
import { type Mode, SLICE_NAME, type StoreState } from "@/session/theme/slice";

const selectMode = (state: StoreState): Mode => state[SLICE_NAME].mode;

export const useSelectMode = (): Mode => Select.useMemo(selectMode, []);

const keyFor = (dark: boolean): string =>
  dark ? Theming.SYNNAX_DARK.key : Theming.SYNNAX_LIGHT.key;

const prefersDark = (): boolean =>
  typeof window?.matchMedia !== "undefined" &&
  window.matchMedia("(prefers-color-scheme: dark)").matches;

const FIXED_KEYS: Record<Exclude<Mode, "system">, string> = {
  light: Theming.SYNNAX_LIGHT.key,
  dark: Theming.SYNNAX_DARK.key,
};

export const useProviderProps = (): Theming.ProviderProps => {
  const mode = useSelectMode();
  const system = mode === "system";
  const [osKey, setOSKey] = useState(() => keyFor(prefersDark()));
  useAsyncEffect(
    async (signal) => {
      if (!system) return;
      if (Runtime.ENGINE !== "tauri") {
        const query = window.matchMedia("(prefers-color-scheme: dark)");
        setOSKey(keyFor(query.matches));
        const handler = (e: MediaQueryListEvent) => setOSKey(keyFor(e.matches));
        query.addEventListener("change", handler);
        return () => query.removeEventListener("change", handler);
      }
      const win = getCurrentWindow();
      const theme = await win.theme();
      if (signal.aborted) return;
      setOSKey(keyFor(theme === "dark"));
      return await win.onThemeChanged(({ payload }) =>
        setOSKey(keyFor(payload === "dark")),
      );
    },
    [system],
  );
  const key = system ? osKey : FIXED_KEYS[mode];
  return useMemo<Theming.ProviderProps>(() => ({ theme: { key } }), [key]);
};
