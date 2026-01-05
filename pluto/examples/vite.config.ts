// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import react from "@vitejs/plugin-react";
import * as path from "path";
import { defineConfig } from "vite";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@synnaxlabs/pluto/dist": path.resolve(__dirname, "../dist"),
      "@synnaxlabs/pluto": path.resolve(__dirname, "../src"),
      "@synnaxlabs/drift/dist": path.resolve(__dirname, "../dist"),
    },
  },
});
