// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Rule } from "eslint";
import fs from "fs";
import path from "path";

const INDEX_FILES = ["index.ts", "index.tsx"];
const CODE_EXTENSIONS = [".ts", ".tsx"];
const TEST_FILE = /\.(spec|test|bench)\.tsx?$/;
const TESTUTIL = "testutil";
const EXTERNAL_FILES = ["external.ts", "external.tsx"];

const isModule = (dir: string): boolean =>
  INDEX_FILES.some((index) => fs.existsSync(path.join(dir, index)));

const isFile = (file: string): boolean =>
  fs.existsSync(file) && fs.statSync(file).isFile();

const contains = (dir: string, file: string): boolean =>
  file.startsWith(`${dir}${path.sep}`);

const findSrc = (file: string): string | null => {
  for (let dir = path.dirname(file); dir !== path.dirname(dir); dir = path.dirname(dir))
    if (fs.existsSync(path.join(dir, "package.json"))) {
      const src = path.join(dir, "src");
      return contains(src, file) ? src : null;
    }
  return null;
};

const resolve = (src: string, importer: string, specifier: string): string | null => {
  let base: string;
  if (specifier.startsWith("@/")) base = path.join(src, specifier.slice(2));
  else if (specifier.startsWith("."))
    base = path.resolve(path.dirname(importer), specifier);
  else return null;
  const ext = path.extname(base);
  const candidates =
    ext === ".js" || ext === ".jsx"
      ? [base.slice(0, -ext.length) + ext.replace("j", "t")]
      : CODE_EXTENSIONS.includes(ext)
        ? [base]
        : [
            ...CODE_EXTENSIONS.map((e) => base + e),
            ...INDEX_FILES.map((index) => path.join(base, index)),
          ];
  return candidates.find(isFile) ?? null;
};

const isBarrel = (file: string): boolean => {
  const name = path.basename(file);
  return INDEX_FILES.includes(name) || name.replace(/\.tsx?$/, "") === TESTUTIL;
};

const isExternal = (file: string): boolean =>
  EXTERNAL_FILES.includes(path.basename(file));

const enclosingModule = (src: string, file: string): string | null => {
  if (!contains(src, file)) return null;
  // An index file under an Astro pages directory is a route, not a barrel.
  const pages = path.join(src, "pages");
  const top = path.dirname(src);
  for (let dir = path.dirname(file); dir !== top; dir = path.dirname(dir))
    if (dir !== pages && !contains(pages, dir) && isModule(dir)) return dir;
  return null;
};

export const barrelImports: Rule.RuleModule = {
  meta: {
    type: "problem",
    docs: {
      description:
        "Import another module through its barrel, and import only barrels in a spec.",
    },
    messages: {
      deep: "Import {{module}} through its barrel, not a file inside it.",
      spec: "A spec imports only module barrels and testutil files, not {{file}}.",
    },
    schema: [],
  },
  create: (context) => {
    const importer = context.filename;
    const src = findSrc(importer);
    if (src == null) return {};
    const spec = TEST_FILE.test(importer);
    const check = (node: Rule.Node, specifier: unknown, merge = false): void => {
      if (typeof specifier !== "string") return;
      const target = resolve(src, importer, specifier);
      if (target == null || isBarrel(target)) return;
      // A Console layer merges another layer's module of the same domain into its own.
      if (merge && isExternal(importer) && isExternal(target)) return;
      const module = enclosingModule(src, target);
      if (module == null) return;
      if (!contains(module, importer))
        context.report({
          node,
          messageId: "deep",
          data: { module: `@/${path.relative(src, module)}` },
        });
      else if (spec)
        context.report({
          node,
          messageId: "spec",
          data: { file: path.relative(path.dirname(importer), target) },
        });
    };
    return {
      ImportDeclaration: (node) => check(node, node.source.value),
      ExportNamedDeclaration: (node) => check(node, node.source?.value),
      ExportAllDeclaration: (node) =>
        check(node, node.source.value, node.exported == null),
      ImportExpression: (node) =>
        check(node, node.source.type === "Literal" ? node.source.value : null),
    };
  },
};
