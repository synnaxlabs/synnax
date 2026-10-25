// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";

import { build } from "vite";
import { afterEach, describe, expect, it } from "vitest";

import { lib, moduleExports, type Options } from "./index.js";

const TSCONFIG = {
  compilerOptions: {
    strict: true,
    target: "ES2022",
    module: "ESNext",
    moduleResolution: "bundler",
    skipLibCheck: true,
    noUncheckedSideEffectImports: false,
    paths: { "@/*": ["./src/*"] },
  },
  include: ["src"],
};

const TSCONFIG_BUILD = {
  extends: "./tsconfig.json",
  compilerOptions: {
    noEmit: false,
    declaration: true,
    emitDeclarationOnly: true,
    rootDir: ".",
  },
};

const TSCONFIG_TRANSPILED = {
  extends: "./tsconfig.json",
  compilerOptions: { noEmit: false, declaration: true, rootDir: "src" },
};

const roots: string[] = [];

const createPackage = (
  files: Record<string, string>,
  { manifest = {}, transpiled = false } = {},
): string => {
  const root = mkdtempSync(path.join(tmpdir(), "vite-plugin-lib-"));
  roots.push(root);
  const all: Record<string, string> = {
    "package.json": JSON.stringify({ type: "module", ...manifest }),
    "tsconfig.json": JSON.stringify(TSCONFIG),
    "tsconfig.build.json": JSON.stringify(
      transpiled ? TSCONFIG_TRANSPILED : TSCONFIG_BUILD,
    ),
    ...files,
  };
  for (const [file, content] of Object.entries(all)) {
    const full = path.join(root, file);
    mkdirSync(path.dirname(full), { recursive: true });
    writeFileSync(full, content);
  }
  return root;
};

const buildPackage = async (root: string, options: Options): Promise<void> => {
  await build({ root, configFile: false, logLevel: "silent", plugins: lib(options) });
};

const emitted = (root: string, file: string): boolean =>
  existsSync(path.join(root, "dist", file));

const read = (root: string, file: string): string =>
  readFileSync(path.join(root, "dist", file), "utf8");

const FLAT = {
  "src/index.ts": 'export * from "@/a";\nexport * from "@/b";\n',
  "src/a.ts": "export const a = 1;\n",
  "src/b.ts": "export const b = 2;\n",
};

afterEach(() => {
  for (const root of roots.splice(0)) rmSync(root, { recursive: true, force: true });
});

describe("lib", () => {
  it("should bundle the package into one file by default", async () => {
    const root = createPackage(FLAT);
    await buildPackage(root, { name: "pkg" });
    expect(emitted(root, "pkg.js")).toBe(true);
    expect(emitted(root, "a.js")).toBe(false);
  });

  describe("transpiled", () => {
    it("should emit one JavaScript and declaration file per source file", async () => {
      const root = createPackage(FLAT, { transpiled: true });
      await buildPackage(root, { name: "pkg", transpiled: true });
      expect(emitted(root, "index.js")).toBe(true);
      expect(emitted(root, "index.d.ts")).toBe(true);
      expect(emitted(root, "a.js")).toBe(true);
      expect(emitted(root, "b.js")).toBe(true);
    });

    it("should rewrite imports to fully specified relative paths", async () => {
      const root = createPackage(
        {
          "src/index.ts": 'export * from "@/a";\nexport * from "./dir";\n',
          "src/a.ts": "export const a = 1;\n",
          "src/dir/index.ts": 'export { a as b } from "@/a";\n',
        },
        { transpiled: true },
      );
      await buildPackage(root, { name: "pkg", transpiled: true });
      expect(read(root, "index.js")).toContain('export * from "./a.js";');
      expect(read(root, "index.js")).toContain('export * from "./dir/index.js";');
      expect(read(root, "dir/index.js")).toContain('from "../a.js";');
      expect(read(root, "index.d.ts")).toContain('export * from "./a.js";');
    });

    it("should copy stylesheets beside their modules", async () => {
      const root = createPackage(
        {
          "src/index.ts": 'import "./button/Button.css";\nexport const a = 1;\n',
          "src/button/Button.css": ".button { color: red; }\n",
        },
        { transpiled: true },
      );
      await buildPackage(root, { name: "pkg", transpiled: true });
      expect(read(root, "button/Button.css")).toEqual(".button { color: red; }\n");
      expect(read(root, "index.js")).toContain('import "./button/Button.css";');
    });
  });

  it("should emit each module entry and its source files as modules", async () => {
    const root = createPackage(
      {
        "src/a/index.ts": 'export * from "@/a/value";\n',
        "src/a/value.ts": "export const a = 1;\n",
        "src/b/index.ts": 'export * from "@/b/value";\n',
        "src/b/value.ts": "export const b = 2;\n",
      },
      { manifest: { exports: moduleExports(["a", "b"]) }, transpiled: true },
    );
    await buildPackage(root, { name: "pkg", modules: true });
    expect(emitted(root, "a/index.js")).toBe(true);
    expect(emitted(root, "a/value.js")).toBe(true);
    expect(emitted(root, "b/index.js")).toBe(true);
  });

  it("should require a static export when the package has a static directory", async () => {
    const files = {
      "src/a/index.ts": "export const a = 1;\n",
      "static/logo.svg": "<svg />\n",
    };
    const missing = createPackage(files, {
      manifest: { exports: moduleExports(["a"]) },
      transpiled: true,
    });
    await expect(buildPackage(missing, { name: "pkg", modules: true })).rejects.toThrow(
      "package.json exports do not match",
    );
    const present = createPackage(files, {
      manifest: { exports: moduleExports(["a"], true) },
      transpiled: true,
    });
    await buildPackage(present, { name: "pkg", modules: true });
    expect(emitted(present, "a/index.js")).toBe(true);
  });

  describe("reactCompiled", () => {
    const REACT = {
      "src/react.d.ts":
        'declare module "react" { export const useState: <T>(v: T) => [T]; }\n' +
        "declare namespace JSX { interface Element {} " +
        "interface IntrinsicElements { [tag: string]: unknown } }\n",
    };
    const HOOK =
      'import { useState } from "react";\n' +
      "export const useCount = () => { const [n] = useState(0); return { n }; };\n";
    const UNCOMPILABLE =
      'import { useState } from "react";\n' +
      "export const useCount = (n?: number) => {\n" +
      "  const [count] = useState(0);\n" +
      "  n ??= count;\n" +
      "  return { n };\n" +
      "};\n";
    const UNCOMPILABLE_ERROR = "Handle ??= operators in AssignmentExpression";
    const COMPILED = 'from "react/compiler-runtime"';

    const buildBundle = async (
      files: Record<string, string>,
      manifest: Record<string, unknown> = {},
    ): Promise<string> => {
      const root = createPackage({ ...REACT, ...files }, { manifest });
      await build({
        root,
        configFile: false,
        logLevel: "silent",
        plugins: lib({ name: "pkg", reactCompiled: true }),
        build: { rolldownOptions: { external: [/^react/] } },
      });
      return read(root, "pkg.js");
    };

    it("should compile each emitted file and replace JSX files", async () => {
      const root = createPackage(
        {
          ...REACT,
          "src/index.ts": 'export * from "@/Count";\n',
          "src/Count.tsx":
            'import { useState } from "react";\n' +
            "export const Count = () => {\n" +
            "  const [n] = useState(0);\n" +
            "  return <p>{n}</p>;\n" +
            "};\n",
        },
        { transpiled: true },
      );
      await buildPackage(root, { name: "pkg", transpiled: true, reactCompiled: true });
      expect(emitted(root, "Count.jsx")).toBe(false);
      expect(read(root, "Count.js")).toContain(COMPILED);
      expect(read(root, "index.js")).toContain('export * from "./Count.js";');
    });

    it("should fail on a hook that the compiler cannot compile", async () => {
      const root = createPackage(
        { ...REACT, "src/index.ts": UNCOMPILABLE },
        { transpiled: true },
      );
      await expect(
        buildPackage(root, { name: "pkg", transpiled: true, reactCompiled: true }),
      ).rejects.toThrow(UNCOMPILABLE_ERROR);
    });

    it("should fail a bundled build on a hook that the compiler cannot compile", async () => {
      await expect(buildBundle({ "src/index.ts": UNCOMPILABLE })).rejects.toThrow(
        UNCOMPILABLE_ERROR,
      );
    });

    it("should keep a callback inside a hook built by a factory", async () => {
      const root = createPackage(
        {
          ...REACT,
          "src/index.ts": 'export * from "@/create";\n',
          "src/create.ts":
            'import { useState } from "react";\n' +
            "export const create = (label: string) => {\n" +
            "  const useLabel = () => {\n" +
            "    const [n] = useState(0);\n" +
            "    return { n, read: () => label };\n" +
            "  };\n" +
            "  return useLabel;\n" +
            "};\n",
        },
        { transpiled: true },
      );
      await buildPackage(root, { name: "pkg", transpiled: true, reactCompiled: true });
      const code = read(root, "create.js");
      expect(code).toContain(COMPILED);
      expect(code).toContain("() => label");
    });

    it("should compile a hook wrapped in a type cast", async () => {
      const code = await buildBundle({
        "src/index.ts":
          'import { useState } from "react";\n' +
          "export const useCount = (() => {\n" +
          "  const [n] = useState(0);\n" +
          "  return { n };\n" +
          "}) as () => { n: number };\n",
      });
      expect(code).toContain(COMPILED);
    });

    it("should leave a lowercase Pluto namespace uncompiled", async () => {
      const code = await buildBundle(
        {
          "src/index.ts": 'export * from "@/telem";\n',
          "src/telem/index.ts": 'export * as telem from "@/telem/count";\n',
          "src/telem/count.ts": HOOK,
        },
        { name: "@synnaxlabs/pluto" },
      );
      expect(code).not.toContain(COMPILED);
    });
  });

  it("should reject a module entry that imports for side effects", async () => {
    const root = createPackage(
      {
        "src/a/index.ts": 'import "./a.css";\nexport const a = 1;\n',
        "src/a/a.css": ".a { color: red; }\n",
      },
      { manifest: { exports: moduleExports(["a"]) }, transpiled: true },
    );
    await expect(buildPackage(root, { name: "pkg", modules: true })).rejects.toThrow(
      "a imports ./a.css",
    );
  });
});
