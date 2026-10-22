// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Query } from "@/components/tabs/sync";

export const PLATFORMS = ["Linux", "Windows", "macOS", "Docker"] as const;

export type Platform = (typeof PLATFORMS)[number];

// The runtime module in @synnaxlabs/x does the same, but it would ship Zod to the page.
const detect = (): Platform | null => {
  const agent = navigator.userAgent.toLowerCase();
  if (agent.includes("mac")) return "macOS";
  if (agent.includes("win")) return "Windows";
  if (agent.includes("linux")) return "Linux";
  return null;
};

/** Shows the reader's own operating system first. */
export const QUERY: Query = { initial: detect };
