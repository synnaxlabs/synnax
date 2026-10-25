// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createRequire } from "node:module";

import { type ParserOptions, transformAsync } from "@babel/core";
import { isWorker, plugins } from "@synnaxlabs/eslint-config/compiler";
import { type Plugin } from "rolldown";

const require = createRequire(import.meta.url);
export const COMPILER = require.resolve("babel-plugin-react-compiler");

export const OPTIONS = {
  // Outlining moves a callback in a hook or component nested in a factory to module
  // scope, where the factory's bindings are unbound (facebook/react#34901).
  environment: { enableFunctionOutlining: false },
  // A compiled worker function throws "Invalid hook call" on the Aether worker.
  sources: (file: string) => !isWorker(file),
};

/** Fails on any component or hook that the compiler cannot compile. */
export const STRICT = { ...OPTIONS, panicThreshold: "all_errors" };

// A workspace dependency's dist is already compiled.
export const DIST = /[/\\]dist[/\\]/;

const SOURCE = /\.[jt]sx?(?:$|\?)/;
const VIRTUAL = /^\0/;
const NODE_MODULES = /[/\\]node_modules[/\\]/;

const PARSER: Record<string, ParserOptions["plugins"]> = {
  ".js": [],
  ".jsx": ["jsx"],
  ".ts": ["typescript"],
  ".tsx": ["typescript", "jsx"],
};

const compile = async (code: string, file: string) =>
  await transformAsync(code, {
    babelrc: false,
    configFile: false,
    filename: file,
    sourceMaps: true,
    parserOpts: {
      sourceType: "module",
      plugins: PARSER[file.slice(file.lastIndexOf("."))],
    },
    plugins: plugins([COMPILER, STRICT]),
  });

/**
 * Compiles each module of a build with the React Compiler on one of Rolldown's worker
 * threads. A module that the compiler cannot compile fails the build.
 */
const thread = (): Plugin => {
  // Rolldown unrefs its worker threads, so a thread that holds no handle of its own
  // exits before the build calls it.
  setInterval(() => {}, 2 ** 30);
  return {
    name: "synnax:react-compiler-thread",
    transform: {
      filter: { id: { include: SOURCE, exclude: [VIRTUAL, NODE_MODULES, DIST] } },
      handler: async (code, id) => {
        const file = id.replace(/\?.*$/, "");
        // The compiler's error reads its message from a getter, which the thread drops
        // when it hands the error to the build.
        const result = await compile(code, file).catch((e: unknown) => {
          throw new Error(e instanceof Error ? e.message : String(e));
        });
        if (result?.code == null)
          throw new Error(`${file}: the React Compiler emitted nothing`);
        return { code: result.code, map: result.map };
      },
    },
  };
};

export default thread;
