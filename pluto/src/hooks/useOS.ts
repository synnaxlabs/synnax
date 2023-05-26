// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { OS } from "@synnaxlabs/x";

export const useOS = (force?: OS, default_: OS | null = null): OS | null =>
  getOS(force, default_);

export const getOS = (force?: OS, default_: OS | null = null): OS | null => {
  if (force != null) return force;
  if (typeof window === "undefined") return null;
  const userAgent = window.navigator.userAgent.toLowerCase();
  if (userAgent.includes("mac")) {
    return "MacOS";
  } else if (userAgent.includes("win")) {
    return "Windows";
  } else if (userAgent.includes("linux")) {
    return "Linux";
  }
  return default_;
};
