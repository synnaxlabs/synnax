// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { invoke } from "@tauri-apps/api/core";
import z from "zod";

/** What this install is, and what this launch knows about the one before it. */
export const installZ = z.object({
  /** Identifies the install across launches, and across a webview data wipe. */
  id: z.string(),
  firstLaunch: z.boolean(),
  hoursSinceLastLaunch: z.number().nullable(),
  os: z.string(),
  arch: z.string(),
});
export interface Install extends z.infer<typeof installZ> {}

export const retrieveInstall = async (): Promise<Install> =>
  installZ.parse(await invoke("install_info"));
