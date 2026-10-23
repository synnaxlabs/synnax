// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { transformFileAsync } from "@babel/core";
import babel, { defineRolldownBabelPreset } from "@rolldown/plugin-babel";
import { existsSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { createRequire } from "node:module";
import path from "node:path";
import { type PluginOption } from "vite";

const require = createRequire(import.meta.url);
const COMPILER = require.resolve("babel-plugin-react-compiler");
const JSX = require.resolve("@babel/plugin-transform-react-jsx");
const MAP_COMMENT = /\n\/\/# sourceMappingURL=\S*\s*$/;

/**
 * Compiles components and hooks with the React Compiler. Unlike the preset that
 * `@vitejs/plugin-react` ships, it applies in every environment, so specs run the same
 * compiled code that builds emit.
 */
export const reactCompiler = (): PluginOption =>
  babel({
    presets: [
      defineRolldownBabelPreset({
        preset: () => ({ plugins: [COMPILER] }),
        rolldown: { optimizeDeps: { include: ["react/compiler-runtime"] } },
      }),
    ],
  });

/**
 * Compiles a JavaScript or JSX file that `tsc` emitted, in place, chaining its source
 * map. A JSX file is replaced by a `.js` file.
 */
export const compileEmitted = async (file: string): Promise<void> => {
  const mapFile = `${file}.map`;
  const mapped = existsSync(mapFile);
  const result = await transformFileAsync(file, {
    babelrc: false,
    configFile: false,
    sourceMaps: mapped,
    inputSourceMap: mapped ? JSON.parse(readFileSync(mapFile, "utf8")) : undefined,
    plugins: [COMPILER, [JSX, { runtime: "automatic" }]],
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
