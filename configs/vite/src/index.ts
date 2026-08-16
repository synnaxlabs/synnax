// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import path from "path";
import dts from "unplugin-dts/vite";
import { type Plugin } from "vite";

export interface Options {
  name: string;
}

export const lib = ({ name }: Options): Plugin[] => {
  const dtsPlugin = dts({});
  return [
    {
      name: "vite-plugin-lib",
      config: (config) => {
        const prod = isProd();
        console.log(
          `\x1b[34m Synnax - ${prod ? "Production" : "Development"} mode\x1b[0m`,
        );
        return {
          resolve: { tsconfigPaths: true },
          build: {
            sourcemap: !prod,
            minify: prod,
            lib: {
              name,
              formats: ["es"],
              fileName: (_, entryName) =>
                `${entryName === "index" ? name : entryName}.js`,
              entry: path.resolve(config.root ?? ".", "src/index.ts"),
              ...config.build?.lib,
            },
          },
        };
      },
    },
    ...(Array.isArray(dtsPlugin) ? dtsPlugin : [dtsPlugin]),
  ];
};

export const isProd = () => process.env.SYNNAX_TS_ENV === "prod";
