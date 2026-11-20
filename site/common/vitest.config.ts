// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import react from "@astrojs/react";
import { getViteConfig } from "astro/config";
import type { ViteUserConfigFn } from "vitest/config";

import { CDN } from "./src/cdn";

const config: ViteUserConfigFn = getViteConfig(
  { test: { environment: "node" } },
  {
    root: import.meta.dirname,
    integrations: [react()],
    image: { domains: [new URL(CDN).hostname] },
  },
);

export default config;
