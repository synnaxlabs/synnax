// Copyright 2026 Synnax Labs, Inc.
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

/**
 * @returns the OS that the user agent names, or undefined outside a browser or when the
 * user agent names none.
 */
export const getOS = (): OS | undefined => {
  if (typeof window === "undefined") return undefined;
  const userAgent = window.navigator.userAgent.toLowerCase();
  if (userAgent.includes("mac")) return "macOS";
  if (userAgent.includes("win")) return "Windows";
  if (userAgent.includes("linux")) return "Linux";
  return undefined;
};
