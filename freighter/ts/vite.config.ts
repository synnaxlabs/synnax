// Copyright 2023 Synnax Labs, Inc.

import { defineConfig } from "vite";

import { lib } from "@synnaxlabs/vite-plugin";

export default defineConfig({
  plugins: [lib({ name: "freighter" })],
});
