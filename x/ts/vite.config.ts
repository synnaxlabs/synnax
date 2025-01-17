// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lib } from "@synnaxlabs/vite-plugin";
import path from "path";
import { defineConfig } from "vite";

export default defineConfig({
  base: "/x/",
  plugins: [lib({ name: "x" })],
  build: {
    lib: {
      entry: {
        index: path.resolve(".", "src/index.ts"),
        binary: path.resolve(".", "src/binary/index.ts"),
        compare: path.resolve(".", "src/compare/index.ts"),
        deep: path.resolve(".", "src/deep/index.ts"),
        kv: path.resolve(".", "src/kv/index.ts"),
        runtime: path.resolve(".", "src/runtime/index.ts"),
        observe: path.resolve(".", "src/observe/index.ts"),
        box: path.resolve(".", "src/spatial/box/index.ts"),
        bounds: path.resolve(".", "src/spatial/bounds/index.ts"),
        dimensions: path.resolve(".", "src/spatial/dimensions/index.ts"),
        direction: path.resolve(".", "src/spatial/direction/index.ts"),
        location: path.resolve(".", "src/spatial/location/index.ts"),
        position: path.resolve(".", "src/spatial/position/index.ts"),
        scale: path.resolve(".", "src/spatial/scale/index.ts"),
        xy: path.resolve(".", "src/spatial/xy/index.ts"),
        spatial: path.resolve(".", "src/spatial/index.ts"),
        telem: path.resolve(".", "src/telem/index.ts"),
        url: path.resolve(".", "src/url/index.ts"),
        worker: path.resolve(".", "src/worker/index.ts"),
        debounce: path.resolve(".", "src/debounce/index.ts"),
        destructor: path.resolve(".", "src/destructor.ts"),
        toArray: path.resolve(".", "src/toArray.ts"),
        search: path.resolve(".", "src/search.ts"),
        unique: path.resolve(".", "src/unique.ts"),
        record: path.resolve(".", "src/record.ts"),
        change: path.resolve(".", "src/change/index.ts"),
        identity: path.resolve(".", "src/identity.ts"),
        caseconv: path.resolve(".", "src/caseconv/index.ts"),
        zodutil: path.resolve(".", "src/zodutil/index.ts"),
      },
    },
    rollupOptions: {
      external: ["zod"],
    },
  },
  test: {
    globals: true,
    environment: "jsdom",
  },
});
