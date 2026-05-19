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
import path from "path";
import { defineConfig, esmExternalRequirePlugin } from "vite";

export default defineConfig({
  base: "/pluto/",
  plugins: [
    esmExternalRequirePlugin({ external: [/^react(-dom)?(\/.*)?$/] }),
    react(),
    lib({ name: "pluto" }),
  ],
  build: {
    lib: {
      entry: {
        access: path.resolve(".", "src/access/index.ts"),
        alamos: path.resolve(".", "src/alamos/index.ts"),
        arc: path.resolve(".", "src/arc/index.ts"),
        axis: path.resolve(".", "src/axis/index.ts"),
        canvas: path.resolve(".", "src/canvas/index.ts"),
        channel: path.resolve(".", "src/channel/index.ts"),
        cluster: path.resolve(".", "src/cluster/index.ts"),
        color: path.resolve(".", "src/color/index.ts"),
        device: path.resolve(".", "src/device/index.ts"),
        diagram: path.resolve(".", "src/diagram/index.ts"),
        direction: path.resolve(".", "src/direction/index.ts"),
        eraser: path.resolve(".", "src/eraser/index.ts"),
        ether: path.resolve(".", "src/ether.ts"),
        flux: path.resolve(".", "src/flux/index.ts"),
        group: path.resolve(".", "src/group/index.ts"),
        input: path.resolve(".", "src/input/index.ts"),
        json: path.resolve(".", "src/json/index.ts"),
        label: path.resolve(".", "src/label/index.ts"),
        legend: path.resolve(".", "src/legend/index.ts"),
        line: path.resolve(".", "src/line/index.ts"),
        lineplot: path.resolve(".", "src/lineplot/index.ts"),
        log: path.resolve(".", "src/log/index.ts"),
        measure: path.resolve(".", "src/measure/index.ts"),
        mosaic: path.resolve(".", "src/mosaic/index.ts"),
        notation: path.resolve(".", "src/notation/index.ts"),
        ontology: path.resolve(".", "src/ontology/index.ts"),
        os: path.resolve(".", "src/os/index.ts"),
        pluto: path.resolve(".", "src/pluto/index.ts"),
        rack: path.resolve(".", "src/rack/index.ts"),
        ranger: path.resolve(".", "src/ranger/index.ts"),
        rule: path.resolve(".", "src/rule/index.ts"),
        schematic: path.resolve(".", "src/schematic/index.ts"),
        status: path.resolve(".", "src/status/index.ts"),
        steps: path.resolve(".", "src/steps/index.ts"),
        synnax: path.resolve(".", "src/synnax/index.ts"),
        table: path.resolve(".", "src/table/index.ts"),
        "table/cells": path.resolve(".", "src/table/cells/index.ts"),
        task: path.resolve(".", "src/task/index.ts"),
        telem: path.resolve(".", "src/telem/index.ts"),
        "telem/aether": path.resolve(".", "src/telem/aether/index.ts"),
        "telem/control": path.resolve(".", "src/telem/control/index.ts"),
        "telem/control/aether": path.resolve(".", "src/telem/control/aether/index.ts"),
        user: path.resolve(".", "src/user/index.ts"),
        value: path.resolve(".", "src/value/index.ts"),
        view: path.resolve(".", "src/view/index.ts"),
        viewport: path.resolve(".", "src/viewport/index.ts"),
        workspace: path.resolve(".", "src/workspace/index.ts"),
      },
    },
    rolldownOptions: {
      external: [
        "react-hook-form",
        "zod",
        "@synnaxlabs/x",
        "@synnaxlabs/lyra",
        "@synnaxlabs/client",
        "@synnaxlabs/alamos",
        "@synnaxlabs/freighter",
        "@synnaxlabs/media",
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
