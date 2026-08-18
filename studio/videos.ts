// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { manifest } from "@/index";

/**
 * The production manifest: one entry per docs video, keyed by the id the docs
 * site's Video component uses. `pnpm batch` produces the light/dark pair for
 * every entry; `pnpm upload` pushes them to the docs CDN.
 */
export default manifest.define([
  { id: "console/line-plots/data-tab", script: "scripts/line-plot-data.ts" },
]);
