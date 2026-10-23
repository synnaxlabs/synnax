// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { existsSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { createRequire } from "node:module";
import path from "node:path";

import { transformFileAsync } from "@babel/core";
import babel, {
  defineRolldownBabelPreset,
  type RolldownBabelPreset,
} from "@rolldown/plugin-babel";
import { type PluginOption } from "vite";

const require = createRequire(import.meta.url);
const COMPILER = require.resolve("babel-plugin-react-compiler");
const JSX = require.resolve("@babel/plugin-transform-react-jsx");
const MAP_COMMENT = /\n\/\/# sourceMappingURL=\S*\s*$/;

/** Fails on any component or hook that the compiler cannot compile. */
const STRICT = { panicThreshold: "all_errors" };

const preset = (build: boolean): RolldownBabelPreset =>
  defineRolldownBabelPreset({
    preset: () => ({ plugins: [[COMPILER, build ? STRICT : {}]] }),
    rolldown: {
      // A workspace dependency's dist is already compiled.
      filter: { id: { exclude: /[/\\]dist[/\\]/ } },
      optimizeDeps: { include: ["react/compiler-runtime"] },
      configResolvedHook: (config) => (config.command === "build") === build,
    },
  });

/**
 * Compiles components and hooks with the React Compiler. Unlike the preset that
 * `@vitejs/plugin-react` ships, it applies in every environment, so specs run the same
 * compiled code that builds emit. A build fails on a component or hook that the
 * compiler cannot compile; a `"use no memo"` directive opts one out.
 */
export const reactCompiler = (): PluginOption =>
  babel({ presets: [preset(true), preset(false)] });

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
    plugins: [[COMPILER, STRICT], [JSX, { runtime: "automatic" }]],
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
