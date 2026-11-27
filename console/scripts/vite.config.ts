// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { defineConfig } from "vite";

// Bundles the preview scripts with their dependencies, so the preview host runs them in
// a Node image without the workspace.
export default defineConfig({
  build: {
    ssr: true,
    target: "node26",
    outDir: "scripts/dist",
    emptyOutDir: true,
    rolldownOptions: {
      input: ["scripts/create_sample.ts", "scripts/create_preview_workspace.ts"],
    },
  },
  ssr: { noExternal: true },
});
