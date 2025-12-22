// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** JavaScript runtime environments. */
export type Runtime = "browser" | "node" | "webworker";

/**
 * Does best effort detection of the runtime environment.
 *
 * @returns The runtime environment.
 */
export const detect = (): Runtime => {
  if (
    typeof process !== "undefined" &&
    process.versions != null &&
    process.versions.node != null
  )
    return "node";

  if (typeof window === "undefined" || window.document === undefined)
    return "webworker";

  return "browser";
};

export const RUNTIME = detect();

/**
 * Detects if the code is running inside a Tauri desktop application.
 *
 * @returns true if running in Tauri, false otherwise.
 */
export const isTauri = (): boolean =>
  typeof window !== "undefined" &&
  "__TAURI_INTERNALS__" in window &&
  window.__TAURI_INTERNALS__ != null;

export const IS_TAURI = isTauri();

declare global {
  interface Window {
    __TAURI_INTERNALS__?: unknown;
  }
}
