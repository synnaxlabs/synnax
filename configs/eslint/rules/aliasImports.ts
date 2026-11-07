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

import { contains, findSrc } from "./packageSrc";

const RELATIVE = /^\.\.?\//;
// Plain Node runs a module that imports with a .ts extension, and Node does not know
// the alias.
const TS_EXTENSION = /\.tsx?$/;

const aliased = new Map<string, boolean>();

const hasAlias = (src: string): boolean => {
  let has = aliased.get(src);
  if (has == null) {
    const tsconfig = path.join(path.dirname(src), "tsconfig.json");
    has =
      fs.existsSync(tsconfig) && fs.readFileSync(tsconfig, "utf8").includes('"@/*"');
    aliased.set(src, has);
  }
  return has;
};

export const aliasImports: Rule.RuleModule = {
  meta: {
    type: "suggestion",
    fixable: "code",
    docs: { description: "Require the @/ alias for an import from the package's src." },
    messages: { relative: "Import {{alias}}, not a relative path." },
    schema: [],
  },
  create: (context) => {
    const importer = context.filename;
    const src = findSrc(importer);
    if (src == null || !hasAlias(src)) return {};
    const check = (node: Rule.Node): void => {
      if (!("source" in node) || node.source?.type !== "Literal") return;
      const { source } = node;
      const specifier = source.value;
      if (typeof specifier !== "string") return;
      if (!RELATIVE.test(specifier) || TS_EXTENSION.test(specifier)) return;
      const target = path.resolve(path.dirname(importer), specifier);
      if (!contains(src, target)) return;
      const alias = `@/${path.relative(src, target).split(path.sep).join("/")}`;
      context.report({
        node: source,
        messageId: "relative",
        data: { alias },
        fix: (fixer) => fixer.replaceText(source, JSON.stringify(alias)),
      });
    };
    return {
      ImportDeclaration: check,
      ExportNamedDeclaration: check,
      ExportAllDeclaration: check,
      ImportExpression: check,
    };
  },
};
