// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnlistenFn } from "@tauri-apps/api/event";
import { getCurrent, onOpenUrl } from "@tauri-apps/plugin-deep-link";

import { Session } from "@/session";

/**
 * Deps are the runtime bindings a deep link hook relies on. They default to the live
 * Tauri deep-link plugin and runtime engine; tests inject fakes to drive links without
 * Tauri.
 */
export interface Deps {
  engine: Session.Runtime.Engine;
  getCurrentURLs: () => Promise<string[] | null>;
  onOpenURL: (handler: (urls: string[]) => void) => Promise<UnlistenFn>;
}

export const DEFAULT_DEPS: Deps = {
  engine: Session.Runtime.ENGINE,
  getCurrentURLs: getCurrent,
  onOpenURL: onOpenUrl,
};
