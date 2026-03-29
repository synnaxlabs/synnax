// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import path from "path";
import { type Plugin } from "vite";
import dts from "vite-plugin-dts";

export interface Options {
  name: string;
}

export const lib = ({ name }: Options): Plugin[] => [
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
            formats: ["es", "cjs"],
            fileName: (format, entryName) => {
              const baseName = entryName === "index" ? name : entryName;
              if (format === "es") return `${baseName}.js`;
              return `${baseName}.${format}`;
            },
            entry: path.resolve(config.root ?? ".", "src/index.ts"),
            ...config.build?.lib,
          },
        },
      };
    },
  },
  dts({}),
];

export const isProd = () => process.env.SYNNAX_TS_ENV === "prod";
