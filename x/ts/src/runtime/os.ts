// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const OPERATING_SYSTEMS = ["macOS", "Windows", "Linux"] as const;
const LOWERCASE_OPERATING_SYSTEMS = ["macos", "windows", "linux"] as const;
const LOWER_TO_UPPER_OPERATING_SYSTEMS: Record<
  (typeof LOWERCASE_OPERATING_SYSTEMS)[number],
  (typeof OPERATING_SYSTEMS)[number]
> = {
  macos: "macOS",
  windows: "Windows",
  linux: "Linux",
};

export const osZ = z
  .enum(OPERATING_SYSTEMS)
  .or(
    z
      .enum(LOWERCASE_OPERATING_SYSTEMS)
      .transform((s) => LOWER_TO_UPPER_OPERATING_SYSTEMS[s]),
  );
export type OS = (typeof OPERATING_SYSTEMS)[number];

const evalOS = (): OS | undefined => {
  if (typeof window === "undefined") return undefined;
  const userAgent = window.navigator.userAgent.toLowerCase();
  if (userAgent.includes("mac")) return "macOS";
  if (userAgent.includes("win")) return "Windows";
  if (userAgent.includes("linux")) return "Linux";
  return undefined;
};

export interface RequiredGetOSProps {
  force?: OS;
  default?: OS;
}

export interface OptionalGetOSProps {
  force?: OS;
  default: OS;
}

export type GetOSProps = RequiredGetOSProps | OptionalGetOSProps;

let os: OS | undefined;

export interface GetOS {
  (props?: RequiredGetOSProps): OS;
  (props?: OptionalGetOSProps): OS | undefined;
}

export const getOS: GetOS = ((props = {}) => {
  const { force, default: default_ } = props;
  if (force != null) return force;
  if (os != null) return os;
  os = evalOS();
  return os ?? default_;
}) as GetOS;

export interface OSInfo {
  hostname: string;
  platform: string;
  arch: string;
  version: string;
}

const BROWSER_OS_INFO: OSInfo = {
  hostname: "Browser",
  platform: evalOS()?.toLowerCase() ?? "unknown",
  arch: "unknown",
  version: "unknown",
};

let cachedOSInfo: OSInfo | null = null;
let osInfoInitPromise: Promise<OSInfo> | null = null;

/**
 * Initializes and caches extended OS information.
 * In Tauri: Uses @tauri-apps/plugin-os for detailed info.
 * In browser: Returns basic info detected from user agent.
 *
 * Safe to call multiple times - subsequent calls return cached value.
 */
export const initOSInfo = async (): Promise<OSInfo> => {
  if (cachedOSInfo != null) return cachedOSInfo;
  if (osInfoInitPromise != null) return osInfoInitPromise;

  osInfoInitPromise = (async () => {
    try {
      const { isTauri } = await import("@tauri-apps/api/core");
      const inTauri = isTauri();
      console.log("[runtime/os] isTauri() returned:", inTauri);
      if (!inTauri) {
        cachedOSInfo = BROWSER_OS_INFO;
        return cachedOSInfo;
      }

      console.log("[runtime/os] Attempting to load OS plugin...");
      const osPlugin = await import("@tauri-apps/plugin-os");
      console.log("[runtime/os] OS plugin loaded, calling functions...");
      const [hostname, platform, arch, version] = await Promise.all([
        osPlugin.hostname(),
        osPlugin.platform(),
        osPlugin.arch(),
        osPlugin.version(),
      ]);
      console.log("[runtime/os] OS info retrieved:", { hostname, platform, arch, version });

      cachedOSInfo = {
        hostname: hostname ?? "Unknown",
        platform,
        arch,
        version,
      };
    } catch (error) {
      // Not in Tauri or plugin not available - use browser fallback
      console.error("[runtime/os] Failed to get OS info:", error);
      cachedOSInfo = BROWSER_OS_INFO;
    }

    return cachedOSInfo;
  })();

  return osInfoInitPromise;
};

/**
 * Gets cached extended OS information synchronously.
 * Returns null if initOSInfo hasn't been called yet.
 */
export const getOSInfo = (): OSInfo | null => cachedOSInfo;

export const getOSInfoAsync = async (): Promise<OSInfo> => initOSInfo();
