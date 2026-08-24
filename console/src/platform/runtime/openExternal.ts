// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { open } from "@tauri-apps/plugin-shell";

import { Session } from "@/session";

/**
 * Opens a URL in the user's browser. Tauri's WebView cannot navigate away from the app,
 * so it hands the URL to the shell plugin instead.
 * @throws {Error} if the platform refuses to open the URL.
 */
export const openExternal = async (url: string): Promise<void> => {
  if (Session.Runtime.ENGINE === "tauri") return await open(url);
  window.open(url, "_blank", "noopener,noreferrer");
};
