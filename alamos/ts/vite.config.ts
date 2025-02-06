// Copyright 2025 Synnax Labs, Inc.
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
  plugins: [lib({ name: "alamos" })],
  build: {
    lib: {
      entry: {
        index: path.resolve(".", "src/index.ts"),
        dev: path.resolve(".", "src/dev/index.ts"),
      },
    },
  },
});
