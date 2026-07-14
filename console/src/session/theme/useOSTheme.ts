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
import { useState } from "react";

import { Runtime } from "@/session/runtime";

const keyFor = (dark: boolean): string =>
  dark ? Theming.SYNNAX_DARK.key : Theming.SYNNAX_LIGHT.key;

const prefersDark = (): boolean =>
  typeof window?.matchMedia !== "undefined" &&
  window.matchMedia("(prefers-color-scheme: dark)").matches;

/**
 * Resolves the theme key matching the operating system color scheme, updating as the OS
 * scheme changes. Subscribes through the Tauri window in the desktop app and falls back
 * to `prefers-color-scheme` on the web. Returns the last resolved key while disabled.
 * @param enabled - When false, stops listening and holds the last resolved key.
 */
export const useOSTheme = (enabled: boolean): string => {
  const [key, setKey] = useState(() => keyFor(prefersDark()));
  useAsyncEffect(
    async (signal) => {
      if (!enabled) return;
      if (Runtime.ENGINE !== "tauri") {
        const query = window.matchMedia("(prefers-color-scheme: dark)");
        setKey(keyFor(query.matches));
        const handler = (e: MediaQueryListEvent) => setKey(keyFor(e.matches));
        query.addEventListener("change", handler);
        return () => query.removeEventListener("change", handler);
      }
      const win = getCurrentWindow();
      setKey(keyFor((await win.theme()) === "dark"));
      if (signal.aborted) return;
      return await win.onThemeChanged(({ payload }) =>
        setKey(keyFor(payload === "dark")),
      );
    },
    [enabled],
  );
  return key;
};
