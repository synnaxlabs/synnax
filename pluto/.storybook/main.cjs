/*
 * Copyright 2023 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

const path = require("path");
const tsconfigPaths = require("vite-tsconfig-paths").default;
console.log(path.resolve(path.dirname(__dirname), "tsconfig.json").toString());
module.exports = {
  stories: ["../src/**/*.stories.mdx", "../src/**/*.stories.@(js|jsx|ts|tsx)"],
  addons: ["@storybook/addon-links", "@storybook/addon-essentials", "@storybook/addon-interactions"],
  framework: {
    name: "@storybook/react-vite",
    options: {}
  },
  features: {
    storyStoreV7: true
  },
  resolve: {},
  async viteFinal(config) {
    config.plugins.push(tsconfigPaths({
      projects: [path.resolve(path.dirname(__dirname), "tsconfig.json")]
    }));
    return config;
  },
  docs: {
    autodocs: true
  }
};