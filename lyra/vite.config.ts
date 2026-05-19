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
import react from "@vitejs/plugin-react";
import { copyFileSync, mkdirSync } from "fs";
import path from "path";
import { defineConfig, esmExternalRequirePlugin } from "vite";

export default defineConfig({
  base: "/lyra/",
  plugins: [
    esmExternalRequirePlugin({ external: [/^react(-dom)?(\/.*)?$/] }),
    react(),
    lib({ name: "lyra" }),
    {
      name: "copy-theme-css",
      closeBundle() {
        mkdirSync("dist", { recursive: true });
        for (const file of ["theme.css", "theme-dark.css", "theme-light.css"]) {
          copyFileSync(
            path.resolve(`src/theming/static/${file}`),
            path.resolve(`dist/${file}`),
          );
        }
      },
    },
  ],
  build: {
    lib: {
      formats: ["es"],
      entry: {
        aether: path.resolve(".", "src/aether/index.ts"),
        "aether-runtime": path.resolve(".", "src/aether/aether/index.ts"),
        "aether-test": path.resolve(".", "src/aether/test/index.ts"),
        breadcrumb: path.resolve(".", "src/breadcrumb/index.ts"),
        button: path.resolve(".", "src/button/index.ts"),
        caret: path.resolve(".", "src/caret/index.ts"),
        component: path.resolve(".", "src/component/index.ts"),
        context: path.resolve(".", "src/context/index.ts"),
        css: path.resolve(".", "src/css/index.ts"),
        cursor: path.resolve(".", "src/cursor/index.ts"),
        dialog: path.resolve(".", "src/dialog/index.ts"),
        divider: path.resolve(".", "src/divider/index.ts"),
        errors: path.resolve(".", "src/errors/index.ts"),
        flex: path.resolve(".", "src/flex/index.ts"),
        form: path.resolve(".", "src/form/index.ts"),
        generic: path.resolve(".", "src/generic/index.ts"),
        haul: path.resolve(".", "src/haul/index.ts"),
        header: path.resolve(".", "src/header/index.ts"),
        hooks: path.resolve(".", "src/hooks/index.ts"),
        icon: path.resolve(".", "src/icon/index.ts"),
        input: path.resolve(".", "src/input/index.ts"),
        key: path.resolve(".", "src/key/index.ts"),
        list: path.resolve(".", "src/list/index.ts"),
        memo: path.resolve(".", "src/memo/index.ts"),
        menu: path.resolve(".", "src/menu/index.ts"),
        nav: path.resolve(".", "src/nav/index.ts"),
        note: path.resolve(".", "src/note/index.ts"),
        portal: path.resolve(".", "src/portal/index.ts"),
        progress: path.resolve(".", "src/progress/index.ts"),
        resize: path.resolve(".", "src/resize/index.ts"),
        select: path.resolve(".", "src/select/index.ts"),
        state: path.resolve(".", "src/state/index.ts"),
        status: path.resolve(".", "src/status/index.ts"),
        "status-aether": path.resolve(".", "src/status/aether/index.ts"),
        store: path.resolve(".", "src/store/index.ts"),
        tabs: path.resolve(".", "src/tabs/index.ts"),
        tag: path.resolve(".", "src/tag/index.ts"),
        telem: path.resolve(".", "src/telem/index.ts"),
        testutil: path.resolve(".", "src/testutil/index.ts"),
        text: path.resolve(".", "src/text/index.ts"),
        "text-base": path.resolve(".", "src/text/base/index.ts"),
        theming: path.resolve(".", "src/theming/index.ts"),
        "theming-aether": path.resolve(".", "src/theming/aether/index.ts"),
        tooltip: path.resolve(".", "src/tooltip/index.ts"),
        tree: path.resolve(".", "src/tree/index.ts"),
        triggers: path.resolve(".", "src/triggers/index.ts"),
        util: path.resolve(".", "src/util/index.ts"),
        video: path.resolve(".", "src/video/index.ts"),
      },
    },
    rolldownOptions: {
      external: [
        /^@synnaxlabs\/(x|alamos|media)(\/.*)?$/,
        /^@fontsource(-variable)?\/.*/,
        "async-mutex",
        "clsx",
        "compromise",
        "compromise-dates",
        "fuse.js",
        "mathjs",
        /^mathjs\/.*/,
        "pluralize",
        "@tanstack/react-virtual",
        "use-sync-external-store",
        "use-sync-external-store/shim",
        "vitest",
        "zod",
        /^zod\/.*/,
      ],
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
    exclude: ["**/node_modules/**", "**/dist/**"],
    coverage: {
      include: ["src/**/*.ts", "src/**/*.tsx"],
      exclude: ["src/**/*.spec.ts", "src/**/*.spec.tsx", "src/**/*.bench.ts"],
    },
  },
});
