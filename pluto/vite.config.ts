// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import path from "path";
import { lib } from "@synnaxlabs/vite-plugin";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [lib({ name: "pluto" })],
  build: {
    sourcemap: true,
    minify: false,
    lib: {
      entry: {
        index: path.resolve(".", "src/index.ts"),
        aether: path.resolve(".", "src/aetherIndex.ts"),
        accordion: path.resolve(".", "src/accordion/index.ts"),
        alamos: path.resolve(".", "src/alamos/index.ts"),
        align: path.resolve(".", "src/align/index.ts"),
        button: path.resolve(".", "src/button/index.ts"),
        channel: path.resolve(".", "src/channel/index.ts"),
        color: path.resolve(".", "src/color/index.ts"),
        css: path.resolve(".", "src/css/index.ts"),
        cursor: path.resolve(".", "src/cursor/index.ts"),
        divider: path.resolve(".", "src/divider/index.ts"),
        dropdown: path.resolve(".", "src/dropdown/index.ts"),
        generic: path.resolve(".", "src/generic/index.ts"),
        haul: path.resolve(".", "src/haul/index.ts"),
        header: path.resolve(".", "src/header/index.ts"),
        hooks: path.resolve(".", "src/hooks/index.ts"),
        input: path.resolve(".", "src/input/index.ts"),
        list: path.resolve(".", "src/list/index.ts"),
        memo: path.resolve(".", "src/memo/index.ts"),
        menu: path.resolve(".", "src/menu/index.ts"),
        mosaic: path.resolve(".", "src/mosaic/index.ts"),
        nav: path.resolve(".", "src/nav/index.ts"),
        os: path.resolve(".", "src/os/index.ts"),
        resize: path.resolve(".", "src/resize/index.ts"),
        select: path.resolve(".", "src/select/index.ts"),
        state: path.resolve(".", "src/state/index.ts"),
        status: path.resolve(".", "src/status/index.ts"),
        synnax: path.resolve(".", "src/synnax/index.ts"),
        tabs: path.resolve(".", "src/tabs/index.ts"),
        tag: path.resolve(".", "src/tag/index.ts"),
        text: path.resolve(".", "src/text/index.ts"),
        theming: path.resolve(".", "src/theming/index.ts"),
        tooltip: path.resolve(".", "src/tooltip/index.ts"),
        tree: path.resolve(".", "src/tree/index.ts"),
        triggers: path.resolve(".", "src/triggers/index.ts"),
        video: path.resolve(".", "src/video/index.ts"),
        viewport: path.resolve(".", "src/viewport/index.ts"),
        "telem/remote": path.resolve(".", "src/telem/remote/index.ts"),
        "telem/control": path.resolve(".", "src/telem/control/index.ts"),
        "telem/core": path.resolve(".", "src/telem/core/index.ts"),
        "telem/noop": path.resolve(".", "src/telem/noop/index.ts"),
        "telem/static": path.resolve(".", "src/telem/remote/index.ts"),
        "vis/axis": path.resolve(".", "src/vis/axis/index.ts"),
        "vis/canvas": path.resolve(".", "src/vis/canvas/index.ts"),
        "vis/draw2d": path.resolve(".", "src/vis/draw2d/index.ts"),
        "vis/line": path.resolve(".", "src/vis/line/index.ts"),
        "vis/lineplot": path.resolve(".", "src/vis/lineplot/index.ts"),
        "vis/measure": path.resolve(".", "src/vis/measure/index.ts"),
        "vis/pid": path.resolve(".", "src/vis/pid/index.ts"),
        "vis/regulator": path.resolve(".", "src/vis/regulator/index.ts"),
        "vis/render": path.resolve(".", "src/vis/render/index.ts"),
        "vis/rule": path.resolve(".", "src/vis/rule/index.ts"),
        "vis/tank": path.resolve(".", "src/vis/tank/index.ts"),
        "vis/tooltip": path.resolve(".", "src/vis/tooltip/index.ts"),
        "vis/value": path.resolve(".", "src/vis/value/index.ts"),
        "vis/valve": path.resolve(".", "src/vis/valve/index.ts"),
      },
    },
    rollupOptions: {
      external: ["react", "react-dom", "react-hook-form"],
      output: {
        globals: {
          react: "React",
          "react-dom": "ReactDOM",
        },
      },
    },
  },
  test: {
    globals: true,
    environment: "jsdom",
    setupFiles: ["src/mock/setuptests.ts"],
  },
});
