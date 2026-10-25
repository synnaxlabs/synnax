// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// <reference types="vitest/config" />

import { createHash } from "node:crypto";
import { existsSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { createRequire } from "node:module";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { transformFileAsync } from "@babel/core";
import babel, {
  defineRolldownBabelPreset,
  type RolldownBabelPreset,
} from "@rolldown/plugin-babel";
import { plugins } from "@synnaxlabs/eslint-config/compiler";
import { defineParallelPlugin } from "rolldown/experimental";
import { type PluginOption, type ResolvedConfig } from "vite";

import { COMPILER, DIST, OPTIONS, STRICT } from "./parallel.js";

const require = createRequire(import.meta.url);
// Worker threads import the module without Vite, so a spec run hands them the source.
const PARALLEL = new URL(
  import.meta.url.endsWith(".ts") ? "parallel.ts" : "parallel.js",
  import.meta.url,
);
const JSX = require.resolve("@babel/plugin-transform-react-jsx");
const MAP_COMMENT = /\n\/\/# sourceMappingURL=\S*\s*$/;
const CACHE_KEY = createHash("sha1")
  .update(readFileSync(new URL(import.meta.url)))
  .update(readFileSync(PARALLEL))
  .update(readFileSync(require.resolve("@synnaxlabs/eslint-config/compiler")))
  .digest("hex");

// Spec files and test helpers never ship, so a spec run compiles them leniently.
const TEST_CODE = /\.spec\.tsx?$|testutil/i;

interface PresetParams {
  strict: boolean;
  applies: (config: ResolvedConfig) => boolean;
  include?: RegExp[];
  exclude?: RegExp[];
}

const preset = ({
  strict,
  applies,
  include,
  exclude = [],
}: PresetParams): RolldownBabelPreset =>
  defineRolldownBabelPreset({
    preset: () => ({
      plugins: plugins([COMPILER, strict ? STRICT : OPTIONS]),
    }),
    rolldown: {
      filter: { id: { include, exclude: [DIST, ...exclude] } },
      optimizeDeps: { include: ["react/compiler-runtime"] },
      configResolvedHook: applies,
    },
  });

const isBuild = (config: ResolvedConfig): boolean => config.command === "build";
const isTest = (config: ResolvedConfig): boolean => config.mode === "test";

const PRESETS = [
  preset({ strict: true, applies: isTest, exclude: [TEST_CODE] }),
  preset({ strict: false, applies: isTest, include: [TEST_CODE] }),
  preset({ strict: false, applies: (c) => !isBuild(c) && !isTest(c) }),
];

/**
 * Compiles components and hooks with the React Compiler. Unlike the preset that
 * `@vitejs/plugin-react` ships, it applies in every environment, so specs run the same
 * compiled code that builds emit. A build or spec run fails on a component or hook that
 * the compiler cannot compile, outside spec files and test helpers; the dev server
 * skips it instead. A `"use no memo"` directive opts one out. Builds compile on
 * Rolldown's worker threads.
 */
export const reactCompiler = (): PluginOption => [
  babel({ presets: PRESETS }),
  {
    ...defineParallelPlugin(fileURLToPath(PARALLEL))({}),
    name: "synnax:react-compiler-build",
    apply: "build",
    enforce: "pre",
  },
  {
    name: "synnax:react-compiler",
    // Vitest keys its module cache on plugin names, not on how they transform.
    configureVitest: ({ defineCacheKeyGenerator }) =>
      defineCacheKeyGenerator(() => CACHE_KEY),
  },
];

/**
 * Compiles a JavaScript or JSX file that `tsc` emitted, in place, chaining its source
 * map. A JSX file is replaced by a `.js` file.
 * @throws if the compiler cannot compile a component or hook in the file.
 */
export const compileEmitted = async (file: string): Promise<void> => {
  const mapFile = `${file}.map`;
  const mapped = existsSync(mapFile);
  const result = await transformFileAsync(file, {
    babelrc: false,
    configFile: false,
    sourceMaps: mapped,
    inputSourceMap: mapped ? JSON.parse(readFileSync(mapFile, "utf8")) : undefined,
    plugins: [...plugins([COMPILER, STRICT]), [JSX, { runtime: "automatic" }]],
  });
  if (result?.code == null)
    throw new Error(`${file}: the React Compiler emitted nothing`);
  const target = file.replace(/\.jsx$/, ".js");
  const name = path.basename(target);
  let code = result.code.replace(MAP_COMMENT, "");
  if (result.map != null) {
    code += `\n//# sourceMappingURL=${name}.map\n`;
    writeFileSync(`${target}.map`, JSON.stringify({ ...result.map, file: name }));
  }
  writeFileSync(target, code);
  if (target === file) return;
  rmSync(file);
  rmSync(mapFile, { force: true });
};
