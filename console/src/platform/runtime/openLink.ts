// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type link } from "@synnaxlabs/x";
import { homeDir, join } from "@tauri-apps/api/path";
import { revealItemInDir } from "@tauri-apps/plugin-opener";

import { openExternal } from "@/platform/runtime/openExternal";
import { Session } from "@/session";

// To open a path can start a program, so the file manager only shows it.
const revealPath = async (path: string): Promise<void> => {
  if (Session.Runtime.ENGINE !== "tauri")
    throw new Error("A browser cannot show a file. Open the log in the Console app.");
  const home = path.startsWith("~/");
  await revealItemInDir(home ? await join(await homeDir(), path.slice(2)) : path);
};

const OPENERS: Record<link.Kind, (target: string) => Promise<void>> = {
  url: openExternal,
  path: revealPath,
};

/**
 * Opens a URL in the browser or its app, and shows a path in the file manager.
 * @throws {Error} if the platform refuses the link, or the path does not exist.
 */
export const openLink = async ({ kind, target }: link.Link): Promise<void> =>
  await OPENERS[kind](target);
