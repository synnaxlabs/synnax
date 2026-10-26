// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Rule } from "eslint";

const HOOK = /^use([A-Z0-9]|$)/;

interface Named {
  type: string;
  name?: string;
  value?: unknown;
  computed?: boolean;
  property?: Named;
}

const nameOf = (node: Named | null | undefined): string | null => {
  if (node == null) return null;
  if (node.type === "Identifier") return node.name ?? null;
  if (node.type === "Literal" && typeof node.value === "string") return node.value;
  if (node.type === "MemberExpression" && node.computed === false)
    return nameOf(node.property);
  return null;
};

/** @returns the name a hook reference is stored under, or null if it is not stored. */
const storedAs = (ref: Rule.Node): string | null => {
  const parent: Rule.Node | null = ref.parent;
  switch (parent?.type) {
    case "VariableDeclarator":
      return parent.init === ref ? nameOf(parent.id) : null;
    case "Property": {
      if (parent.computed) return null;
      if (parent.parent.type !== "ObjectPattern")
        return parent.value === ref ? nameOf(parent.key) : null;
      if (parent.key !== ref) return null;
      const { value } = parent;
      return nameOf(value.type === "AssignmentPattern" ? value.left : value);
    }
    case "AssignmentExpression":
    case "AssignmentPattern":
      return parent.right === ref ? nameOf(parent.left) : null;
    case "ImportSpecifier":
      return parent.imported === ref ? nameOf(parent.local) : null;
    case "ExportSpecifier":
      return parent.local === ref ? nameOf(parent.exported) : null;
    default:
      return null;
  }
};

export const namedHookValues: Rule.RuleModule = {
  meta: {
    type: "problem",
    docs: {
      description:
        "Require a hook stored as a value to keep a hook name, so the React Compiler " +
        "calls it as a hook.",
    },
    messages: {
      renamed:
        "Store {{hook}} under a name that starts with use: the React Compiler knows a " +
        "hook only by its name, so it caches a call to {{name}} like a plain function.",
    },
    schema: [],
  },
  create: (context) => {
    const check = (ref: Rule.Node): void => {
      const hook = nameOf(ref);
      if (hook == null || !HOOK.test(hook)) return;
      const name = storedAs(ref);
      if (name == null || HOOK.test(name)) return;
      context.report({ node: ref, messageId: "renamed", data: { hook, name } });
    };
    return {
      Identifier: (node) => {
        if (node.parent.type !== "MemberExpression" || node.parent.object === node)
          check(node);
      },
      MemberExpression: check,
    };
  },
};
