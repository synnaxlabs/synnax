// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Tabs } from "@/components/tabs";

export const PLATFORMS = ["Linux", "Windows", "macOS", "Docker"] as const;

export type Platform = (typeof PLATFORMS)[number];

/**
 * Shows the reader's own operating system first, as the page head marks it on the root,
 * and marks the platform the reader picks, so keys on the page follow it. Docker runs
 * on any system, so it leaves the mark alone.
 */
export const QUERY: Tabs.Query = {
  initial: () =>
    PLATFORMS.find((platform) => platform === document.documentElement.dataset.os) ??
    null,
  onChange: (platform) => {
    if (platform !== "Docker") document.documentElement.dataset.os = platform;
  },
};
