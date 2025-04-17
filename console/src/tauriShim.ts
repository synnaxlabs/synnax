// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { isTauri } from "@tauri-apps/api/core";
import { getCurrentWindow as tauriGetCurrentWindow, type Window } from "@tauri-apps/api/window";

/**
 * An object that looks like `Window` from `@tauri-apps/api/window`.
 */
type WindowLike = Pick<Window, "label" | "show" | "close" | "minimize" | "maximize" | "onThemeChanged" | "theme">;

/**
 * Get the current Tauri window, or a similarly-shaped stub object if running in the browser.
 */
export const getCurrentWindow = (): Window | WindowLike => {
    if (isTauri())
        return tauriGetCurrentWindow();

    return {
        label: MAIN_WINDOW,
        show: async () => { },
        close: async () => { },
        minimize: async () => { },
        maximize: async () => { },
        onThemeChanged: async (_handler) => () => { },
        theme: async () => null,
    }
};