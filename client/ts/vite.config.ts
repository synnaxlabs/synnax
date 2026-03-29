// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// <reference types="vitest/config" />

import { lib } from "@synnaxlabs/vite-plugin";
import { defineConfig } from "vite";

import packageJSON from "./package.json";

export default defineConfig({
  define: { __VERSION__: JSON.stringify(packageJSON.version) },
  plugins: [lib({ name: "client" })],
  build: { rollupOptions: { external: ["zod"] } },
  test: { globals: true, environment: "jsdom" },
});
