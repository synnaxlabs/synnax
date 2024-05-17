// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import * as path from "path";

import type { ConfigEnv, UserConfig } from "vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import tsconfigPaths from "vite-tsconfig-paths";
import { pluginExposeRenderer } from "./vite.base.config";

const isDev = process.env.TAURI_DEBUG === "true";

// eslint-disable-next-line import/no-default-export
export default defineConfig((env) => {
  const forgeEnv = env as ConfigEnv<"renderer">;
  const { root, mode, forgeConfigSelf } = forgeEnv;
  const name = forgeConfigSelf.name ?? "";
  return {
    clearScreen: false,
    server: {
      port: 5173,
      strictPort: true,
    },
    resolve: {
      // alias: {
      //   "@synnaxlabs/pluto/dist": path.resolve(__dirname, "../pluto/dist"),
      //   "@synnaxlabs/pluto": path.resolve(__dirname, "../pluto/src"),
      // },
    },
    envPrefix: ["VITE_", "TAURI_"],
    plugins: [react(), tsconfigPaths(), pluginExposeRenderer(name)],
    build: {
      target: process.env.TAURI_PLATFORM === "windows" ? "chrome105" : "safari16",
      minify: isDev ? "esbuild" : false,
      sourcemap: isDev,
      // We don't really care about maintaining a small bundle size right now, as this file
      // is loaded directly from disc instead of OTN
      chunkSizeWarningLimit: 10000 /* kbs */,
    },
  };
});
