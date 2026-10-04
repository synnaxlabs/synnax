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
