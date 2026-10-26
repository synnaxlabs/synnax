// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";

import { transformSync } from "@babel/core";
import compiler from "babel-plugin-react-compiler";
import { afterEach, describe, expect, it } from "vitest";

import { isWorker, plugins } from "./compiler";

const compile = (code: string): string => {
  const result = transformSync(code, {
    filename: "file.tsx",
    babelrc: false,
    configFile: false,
    parserOpts: { plugins: ["typescript", "jsx"] },
    plugins: plugins([compiler, { panicThreshold: "all_errors" }]),
  });
  if (result?.code == null) throw new Error("the compiler emitted nothing");
  return result.code;
};

describe("plugins", () => {
  describe("bare use hooks", () => {
    it("should compile a hook named use and export it as use", () => {
      const code = compile(
        'import { useState } from "react";\n' +
          "export const use = () => { const [n] = useState(0); return { n }; };\n",
      );
      expect(code).toContain("react/compiler-runtime");
      expect(code).toContain("const use = () => {");
      expect(code).toContain("export { use };");
    });

    it("should call a namespace use hook on every render", () => {
      const code = compile(
        'import { Synnax } from "@synnaxlabs/pluto";\n' +
          "export const C = () => {\n" +
          "  const c = Synnax.use();\n" +
          "  return <p onClick={() => c.k}>x</p>;\n" +
          "};\n",
      );
      // At the top of the body, outside the block that runs only once.
      expect(code).toContain("\n  const c = Synnax.use();\n");
    });

    it("should call an imported use hook on every render", () => {
      const code = compile(
        'import { use } from "./hooks";\n' +
          "export const C = ({ k }: { k: string }) => <p>{use(k)}</p>;\n",
      );
      expect(code).toContain("const t1 = use(k);");
      expect(code).toContain('import { use } from "./hooks";');
    });

    it("should reject a conditional call to a use hook", () => {
      expect(() =>
        compile(
          'import { Synnax } from "@synnaxlabs/pluto";\n' +
            "export const C = ({ on }: { on: boolean }) => {\n" +
            "  const c = on ? Synnax.use() : null;\n" +
            "  return <p>{c?.k}</p>;\n" +
            "};\n",
        ),
      ).toThrow("Hooks must always be called in a consistent order");
    });

    it("should reject a destructured use hook called in a callback", () => {
      expect(() =>
        compile(
          'import { useState } from "react";\n' +
            "export const useA = (xs: { use: () => number }[]) => {\n" +
            "  useState(0);\n" +
            "  return xs.map(({ use }) => use());\n" +
            "};\n",
        ),
      ).toThrow("Hooks must be called at the top level");
    });

    it("should restore a destructured use key", () => {
      expect(
        compile(
          "export const f = ({ use }: { use: number }) => use;\n" +
            "export const g = ({ use: n }: { use: number }) => n;\n",
        ),
      ).toContain("({\n  use\n}: {\n  use: number;\n}) => use;");
    });

    it("should allow a conditional call to React's use", () => {
      const code = compile(
        'import * as React from "react";\n' +
          'import { use } from "react";\n' +
          "export const C = ({ p }: { p?: Promise<string> }) => {\n" +
          "  if (p == null) return null;\n" +
          "  return <p>{use(p)}{React.use(p)}</p>;\n" +
          "};\n",
      );
      expect(code).toContain('import { use } from "react";');
      expect(code).toContain("React.use(p)");
    });

    it("should keep a use method and a use key unchanged", () => {
      const code = compile(
        "class Client { use(m: number) { return m; } }\n" +
          "export const x = { use: 1 };\n" +
          "export const n = new Client().use(x.use);\n",
      );
      expect(code).toContain("use(m: number)");
      expect(code).toContain("use: 1");
      expect(code).toContain("new Client().use(x.use)");
    });

    it("should reject a file that names a variable useBare$", () => {
      expect(() => compile("export const useBare$ = 1;\n")).toThrow(
        "useBare$ is reserved for the React Compiler",
      );
    });

    it("should accept useBare$ in a string", () => {
      expect(compile('export const name = "useBare$";\n')).toContain('"useBare$"');
    });
  });

  describe("hooks and memo components", () => {
    it("should compile a hook that calls no other hook", () => {
      const code = compile(
        "export const useDouble = (n: number) => ({ value: n * 2 });\n",
      );
      expect(code).toContain("const $ = _c(2);");
    });

    it("should compile a hook that a factory returns in an object", () => {
      const code = compile(
        "export const create = (k: number) => ({\n" +
          "  useValue: (n: number) => ({ n, k }),\n" +
          "});\n",
      );
      expect(code).toContain("useValue: n => {");
      expect(code).toContain("const $ = _c(2);");
    });

    it("should compile a memo component that renders no JSX", () => {
      const code = compile(
        'import { memo } from "react";\n' +
          'export const C = memo(({ a }: { a: string[] }) => a.join(","));\n',
      );
      expect(code).toContain("const $ = _c(2);");
    });

    it("should leave a hook with a use no memo directive uncompiled", () => {
      const code = compile(
        "export const useDouble = (n: number) => {\n" +
          '  "use no memo";\n' +
          "  return { value: n * 2 };\n" +
          "};\n",
      );
      expect(code).not.toContain("react/compiler-runtime");
    });

    it("should leave a file with a use no memo directive uncompiled", () => {
      const code = compile(
        '"use no memo";\n' +
          "export const useDouble = (n: number) => ({ value: n * 2 });\n",
      );
      expect(code).not.toContain("react/compiler-runtime");
    });

    it("should leave a function not named like a hook uncompiled", () => {
      const code = compile(
        "export const double = (n: number) => ({ value: n * 2 });\n",
      );
      expect(code).not.toContain("react/compiler-runtime");
    });
  });

  describe("a local Symbol", () => {
    it("should compile a component named Symbol", () => {
      const code = compile(
        'import { Edit } from "./edit";\n' +
          "export const Symbol = () => <p><Edit /></p>;\n",
      );
      expect(code).toContain("const Symbol = () => {\n  const $ = _c(1);");
      expect(code).toContain('globalThis.Symbol.for("react.memo_cache_sentinel")');
    });

    it("should leave an imported Symbol unchanged", () => {
      const code = compile(
        'import { Symbol } from "./symbol";\n' +
          'export const C = () => <p><b>{Symbol.label("a")}</b></p>;\n',
      );
      expect(code).toContain('import { Symbol } from "./symbol";');
      expect(code).toContain('globalThis.Symbol.for("react.memo_cache_sentinel")');
    });
  });
});

describe("isWorker", () => {
  const PLUTO = "@synnaxlabs/pluto";
  const roots: string[] = [];

  afterEach(() => {
    for (const root of roots.splice(0)) rmSync(root, { recursive: true, force: true });
  });

  const createPackage = (name: string, files: Record<string, string>): string => {
    const root = mkdtempSync(path.join(tmpdir(), "is-worker-"));
    roots.push(root);
    for (const [file, content] of Object.entries({
      "package.json": JSON.stringify({ name }),
      ...files,
    })) {
      mkdirSync(path.join(root, path.dirname(file)), { recursive: true });
      writeFileSync(path.join(root, file), content);
    }
    return root;
  };

  const namespace = (name: string): string =>
    `export * as ${name} from "./external";\n`;

  it("should report a file below a lowercase Pluto namespace", () => {
    const root = createPackage(PLUTO, { "src/telem/index.ts": namespace("telem") });
    expect(isWorker(path.join(root, "src/telem/source/remote.ts"))).toBe(true);
  });

  it("should not report a PascalCase namespace in an aether directory", () => {
    const root = createPackage(PLUTO, { "src/aether/index.ts": namespace("Aether") });
    expect(isWorker(path.join(root, "src/aether/store.ts"))).toBe(false);
  });

  it("should read the namespace from the nearest barrel", () => {
    const root = createPackage(PLUTO, {
      "src/telem/index.ts": namespace("Telem"),
      "src/telem/aether/index.ts": namespace("telem"),
    });
    expect(isWorker(path.join(root, "src/telem/aether/remote.ts"))).toBe(true);
    expect(isWorker(path.join(root, "src/telem/Provider.tsx"))).toBe(false);
  });

  it("should not report a file outside any namespace", () => {
    const root = createPackage(PLUTO, { "src/index.ts": 'export * from "./a";\n' });
    expect(isWorker(path.join(root, "src/vis/draw2d/canvas.ts"))).toBe(false);
  });

  it("should not report a lowercase namespace outside Pluto", () => {
    const root = createPackage("@synnaxlabs/lyra", {
      "src/state/index.ts": namespace("state"),
    });
    expect(isWorker(path.join(root, "src/state/state.ts"))).toBe(false);
  });
});
