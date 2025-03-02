// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// <reference types="vitest/config" />

import { lib } from "@synnaxlabs/vite-plugin";
import path from "path";
import { defineConfig } from "vite";

export default defineConfig({
  base: "/pluto/",
  plugins: [lib({ name: "pluto" })],
  build: {
    lib: {
      entry: {
        index: path.resolve(".", "src/index.ts"),
        ether: path.resolve(".", "src/ether.ts"),
        tabs: path.resolve(".", "src/tabs/index.ts"),
        theming: path.resolve(".", "src/theming/index.ts"),
        menu: path.resolve(".", "src/menu/index.ts"),
        header: path.resolve(".", "src/header/index.ts"),
        align: path.resolve(".", "src/align/index.ts"),
        tree: path.resolve(".", "src/tree/index.ts"),
        dropdown: path.resolve(".", "src/dropdown/index.ts"),
        button: path.resolve(".", "src/button/index.ts"),
        video: path.resolve(".", "src/video/index.ts"),
        text: path.resolve(".", "src/text/index.ts"),
        input: path.resolve(".", "src/input/index.ts"),
        triggers: path.resolve(".", "src/triggers/index.ts"),
        list: path.resolve(".", "src/list/index.ts"),
        css: path.resolve(".", "src/css/index.ts"),
        color: path.resolve(".", "src/color/index.ts"),
      },
    },
    rollupOptions: {
      external: ["react", "react-dom", "react-hook-form", "zod"],
      output: {
        globals: {
          react: "React",
          "react-dom": "ReactDOM",
        },
        preserveModules: false,
      },
    },
  },
  test: {
    globals: true,
    environment: "jsdom",
    setupFiles: ["src/mock/setuptests.ts"],
  },
});
