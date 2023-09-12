// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export const OPERATING_SYSTEMS = ["MacOS", "Windows", "Linux", "Docker"] as const;
export type OS = (typeof OPERATING_SYSTEMS)[number];

export interface GetOSProps {
  force?: OS;
  default?: OS;
}

let os: OS | undefined;

const evalOS = (): OS | undefined => {
  if (typeof window === "undefined") return undefined;
  const userAgent = window.navigator.userAgent.toLowerCase();
  if (userAgent.includes("mac")) return "MacOS";
  else if (userAgent.includes("win")) return "Windows";
  else if (userAgent.includes("linux")) return "Linux";
  return undefined;
};

export const getOS = (props: GetOSProps = {}): OS | undefined => {
  const { force, default: default_ } = props;
  if (force != null) return force;
  if (os != null) return os;
  os = evalOS();
  return os ?? default_;
};
