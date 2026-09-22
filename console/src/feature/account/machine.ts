// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { hostname } from "@tauri-apps/plugin-os";

import { Session } from "@/session";

/** The name a machine gets when the platform reports none. */
export const DEFAULT_MACHINE_NAME = "Synnax Desktop";

/** The name this machine shows in the portal: its hostname when known. */
export const readMachineName = async (): Promise<string> => {
  if (Session.Runtime.ENGINE !== "tauri") return DEFAULT_MACHINE_NAME;
  const name = (await hostname())?.trim();
  return name == null || name === "" ? DEFAULT_MACHINE_NAME : name;
};
