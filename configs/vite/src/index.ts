// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { promises as fs } from "fs";
import path from "path";
import ts, { type CompilerOptions } from "typescript";
import { type Alias, type AliasOptions, type Plugin } from "vite";
import dts from "vite-plugin-dts";

const { parseConfigFileTextToJson, parseJsonConfigFileContent, sys } = ts;

const logError = (text: string): void => console.error(`[vite:synnax] ${text}`);

export interface Options {
  name: string;
}

export const tsConfigPaths = ({ name }: Options): Plugin => ({
  name: "vite-plugin-lib:alias",
  enforce: "pre",
  config: async (config) => {
    const prod = isProd();
    console.log(`\x1b[34m Synnax - ${prod ? "Production" : "Development"} mode\x1b[0m`);
    const tsconfigPath = path.resolve(config.root ?? ".", "tsconfig.json");
    const { baseUrl, paths } = await readConfig(tsconfigPath);
    if (baseUrl == null || paths == null) return config;
    const aliasOptions: Alias[] = Object.entries(paths).map(([alias, replacement]) => ({
      find: alias.replace("/*", ""),
      replacement: path.resolve(
        tsconfigPath,
        baseUrl,
        replacement[0].replace("/*", ""),
      ),
    }));
    const existingAlias = transformExistingAlias(config.resolve?.alias);
    return {
      ...config,
      resolve: { ...config.resolve, alias: [...existingAlias, ...aliasOptions] },
      build: {
        sourcemap: !isProd(),
        minify: isProd(),
        lib: {
          name,
          formats: ["es", "cjs"],
          fileName: (format) => {
            if (format === "es") return `${name}.js`;
            return `${name}.${format}`;
          },
          entry: path.resolve(config.root ?? ".", "src/index.ts"),
          ...config.build?.lib,
        },
        ...config.build,
      },
    };
  },
});

export const lib = (options: Options): Plugin[] => [tsConfigPaths(options), dts({})];

const transformExistingAlias = (alias: AliasOptions | undefined): Alias[] => {
  if (alias == null) return [];
  if (Array.isArray(alias)) return alias;
  return Object.entries(alias).map(([find, replacement]) => ({ find, replacement }));
};

const readConfig = async (configPath: string): Promise<CompilerOptions> => {
  try {
    const configFileText = await fs.readFile(configPath, { encoding: "utf-8" });
    const { config } = parseConfigFileTextToJson(configPath, configFileText);
    const { options } = parseJsonConfigFileContent(
      config,
      sys,
      path.dirname(configPath),
    );
    return options;
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : String(error);
    logError(`Could not read tsconfig.json: ${message}.`);
    return {};
  }
};

export const isProd = () => process.env.SYNNAX_TS_ENV === "prod";
