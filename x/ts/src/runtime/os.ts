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
