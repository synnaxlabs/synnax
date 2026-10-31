// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { mkdirSync, mkdtempSync, realpathSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";

import { createServer, type ViteDevServer } from "vite";
import { afterEach, describe, expect, it } from "vitest";

import { reactCompiler } from "./index.js";

const UNCOMPILABLE =
  'import { useState } from "react";\n' +
  "export const useCount = (n?: number) => {\n" +
  "  const [count] = useState(0);\n" +
  "  n ??= count;\n" +
  "  return { n };\n" +
  "};\n";

const COMPILED = "compiler-runtime";

// Lets the dev server resolve the imports that source and compiled code make.
const REACT_STUB = {
  "node_modules/react/package.json": JSON.stringify({
    name: "react",
    exports: { ".": "./index.js", "./compiler-runtime": "./compiler-runtime.js" },
  }),
  "node_modules/react/index.js": "export const useState = (v) => [v];\n",
  "node_modules/react/compiler-runtime.js": "export const c = (n) => new Array(n);\n",
};

const servers: ViteDevServer[] = [];
const roots: string[] = [];

afterEach(async () => {
  await Promise.all(servers.splice(0).map(async (s) => await s.close()));
  for (const root of roots.splice(0)) rmSync(root, { recursive: true, force: true });
});

const transform = async (
  mode: string,
  file: string,
  source: string,
): Promise<string | undefined> => {
  // Windows runners give tmpdir() an 8.3 short name, which Vite refuses to load from.
  const root = realpathSync.native(mkdtempSync(path.join(tmpdir(), "react-compiler-")));
  roots.push(root);
  const files = { ...REACT_STUB, [file]: source };
  for (const [name, content] of Object.entries(files)) {
    mkdirSync(path.join(root, path.dirname(name)), { recursive: true });
    writeFileSync(path.join(root, name), content);
  }
  const server = await createServer({
    root,
    mode,
    configFile: false,
    logLevel: "silent",
    plugins: [reactCompiler()],
    server: { middlewareMode: true, ws: false },
    optimizeDeps: { noDiscovery: true, include: [] },
  });
  servers.push(server);
  const result = await server.transformRequest(`/${file}`);
  return result?.code;
};

describe("reactCompiler", () => {
  it("should fail a spec run on a hook that the compiler cannot compile", async () => {
    await expect(transform("test", "src/useCount.ts", UNCOMPILABLE)).rejects.toThrow(
      "Handle ??= operators in AssignmentExpression",
    );
  });

  it("should skip a hook it cannot compile in a spec file", async () => {
    const code = await transform("test", "src/useCount.spec.ts", UNCOMPILABLE);
    expect(code).not.toContain(COMPILED);
  });

  it("should skip a hook it cannot compile on the dev server", async () => {
    const code = await transform("development", "src/useCount.ts", UNCOMPILABLE);
    expect(code).not.toContain(COMPILED);
  });

  it("should compile a hook named use", async () => {
    const code = await transform(
      "test",
      "src/use.ts",
      'import { useState } from "react";\n' +
        "export const use = () => { const [n] = useState(0); return { n }; };\n",
    );
    expect(code).toContain(COMPILED);
  });

  it("should compile a hook in a spec run", async () => {
    const code = await transform(
      "test",
      "src/useCount.ts",
      'import { useState } from "react";\n' +
        "export const useCount = () => { const [n] = useState(0); return { n }; };\n",
    );
    expect(code).toContain(COMPILED);
  });
});
