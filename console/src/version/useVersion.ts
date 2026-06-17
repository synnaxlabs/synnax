// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax, useAsyncEffect } from "@synnaxlabs/pluto";
import { getVersion } from "@tauri-apps/api/app";
import { useState } from "react";

import { Runtime } from "@/runtime";

/**
 * Returns the version to display for the running Console. When the Console is served by
 * an embedded cluster (web build), this is the connected node's version; otherwise it is
 * the standalone Tauri application version.
 */
export const useVersion = (): string => {
  const connState = Synnax.useConnectionState();
  const [appVersion, setAppVersion] = useState("");
  useAsyncEffect(async (signal) => {
    if (Runtime.ENGINE !== "tauri") return;
    const version = await getVersion();
    if (signal.aborted) return;
    setAppVersion(version);
  }, []);
  if (Runtime.ENGINE !== "tauri") return connState.nodeVersion ?? "";
  return appVersion;
};
