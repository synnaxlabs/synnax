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

import { type NodePath, type PluginObj, transformFileAsync } from "@babel/core";
import babel, {
  defineRolldownBabelPreset,
  type RolldownBabelPreset,
} from "@rolldown/plugin-babel";
import { type PluginOption } from "vite";

const require = createRequire(import.meta.url);
const COMPILER = require.resolve("babel-plugin-react-compiler");
const JSX = require.resolve("@babel/plugin-transform-react-jsx");
const MAP_COMMENT = /\n\/\/# sourceMappingURL=\S*\s*$/;
const CACHE_KEY = createHash("sha1")
  .update(readFileSync(new URL(import.meta.url)))
  .digest("hex");

// Outlining moves a callback in a hook or component nested in a factory to module
// scope, where the factory's bindings are unbound.
const OPTIONS = { environment: { enableFunctionOutlining: false } };

/**
 * Renames each binding that shadows the global `Symbol`, such as a `Symbol` namespace
 * import. The compiler reads its memo sentinel through a bare `Symbol.for`.
 */
const UNSHADOW_SYMBOL: PluginObj = {
  visitor: {
    Program: (program) => {
      const rename = (scoped: NodePath) => {
        if (scoped.scope.hasOwnBinding("Symbol")) scoped.scope.rename("Symbol");
      };
      rename(program);
      program.traverse({ Scopable: rename });
    },
  },
};

/**
 * Removes a TypeScript cast around a function, which hides a component or hook from the
 * compiler. `tsc` output has no casts, so source and emitted builds compile alike.
 */
const UNWRAP_CASTS: PluginObj = {
  visitor: {
    Program: (program) =>
      program.traverse({
        "TSAsExpression|TSSatisfiesExpression": (cast: NodePath) => {
          const inner = cast.get("expression") as NodePath;
          if (inner.isFunction()) cast.replaceWith(inner.node);
        },
      }),
  },
};

/** Fails on any component or hook that the compiler cannot compile. */
const STRICT = { ...OPTIONS, panicThreshold: "all_errors" };

// A workspace dependency's dist is already compiled.
const DIST = /[/\\]dist[/\\]/;

const preset = (build: boolean, exclude: RegExp[]): RolldownBabelPreset =>
  defineRolldownBabelPreset({
    preset: () => ({
      plugins: [UNSHADOW_SYMBOL, UNWRAP_CASTS, [COMPILER, build ? STRICT : OPTIONS]],
    }),
    rolldown: {
      filter: { id: { exclude: [DIST, ...exclude] } },
      optimizeDeps: { include: ["react/compiler-runtime"] },
      configResolvedHook: (config) => (config.command === "build") === build,
    },
  });

export interface ReactCompilerOptions {
  /**
   * Modules the compiler leaves as written, such as code that runs outside React but
   * names its functions like hooks.
   */
  exclude?: RegExp[];
}

/**
 * Compiles components and hooks with the React Compiler. Unlike the preset that
 * `@vitejs/plugin-react` ships, it applies in every environment, so specs run the same
 * compiled code that builds emit. A build fails on a component or hook that the
 * compiler cannot compile; a `"use no memo"` directive opts one out.
 */
export const reactCompiler = ({
  exclude = [],
}: ReactCompilerOptions = {}): PluginOption => [
  babel({ presets: [preset(true, exclude), preset(false, exclude)] }),
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
    plugins: [UNSHADOW_SYMBOL, [COMPILER, STRICT], [JSX, { runtime: "automatic" }]],
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
