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
const NAMED = /^(use[A-Z0-9]|[A-Z])/;
const WRAPPERS = new Set(["memo", "forwardRef"]);
const CASTS = new Set([
  "TSAsExpression",
  "TSSatisfiesExpression",
  "TSNonNullExpression",
  "TSTypeAssertion",
]);
const FUNCTIONS = new Set([
  "ArrowFunctionExpression",
  "FunctionExpression",
  "FunctionDeclaration",
]);

const calleeName = (callee: Rule.Node): string | null => {
  if (callee.type === "Identifier") return callee.name;
  if (
    callee.type === "MemberExpression" &&
    !callee.computed &&
    callee.property.type === "Identifier"
  )
    return callee.property.name;
  return null;
};

const isWrapperCall = (node: Rule.Node): boolean =>
  node.type === "CallExpression" &&
  WRAPPERS.has(calleeName(node.callee as Rule.Node) ?? "");

/** @returns the name a binding gives the function, through memo, forwardRef, and casts. */
const nameOf = (fn: Rule.Node): string | null => {
  if (
    (fn.type === "FunctionDeclaration" || fn.type === "FunctionExpression") &&
    fn.id != null
  )
    return fn.id.name;
  let node: Rule.Node = fn;
  let parent: Rule.Node | null = fn.parent;
  while (parent != null && (CASTS.has(parent.type) || isWrapperCall(parent))) {
    node = parent;
    parent = parent.parent;
  }
  if (parent?.type === "VariableDeclarator" && parent.id.type === "Identifier")
    return parent.id.name;
  if (
    parent?.type === "AssignmentExpression" &&
    parent.right === node &&
    parent.left.type === "Identifier"
  )
    return parent.left.name;
  return null;
};

/** A `useX: () => useY(...)` member only forwards a hook, so it has nothing to cache. */
const isForwardingMember = (fn: Rule.Node): boolean => {
  const parent: Rule.Node | null = fn.parent;
  return (
    parent?.type === "Property" &&
    parent.key.type === "Identifier" &&
    NAMED.test(parent.key.name) &&
    fn.type === "ArrowFunctionExpression" &&
    fn.body.type === "CallExpression"
  );
};

const insideComponentOrHook = (fn: Rule.Node): boolean => {
  for (let node: Rule.Node | null = fn.parent; node != null; node = node.parent)
    if (FUNCTIONS.has(node.type)) return NAMED.test(nameOf(node) ?? "");
  return false;
};

export const namedHookCallers: Rule.RuleModule = {
  meta: {
    type: "problem",
    docs: {
      description:
        "Require a component or hook name on a function that calls a hook, so the " +
        "React Compiler compiles it.",
    },
    messages: {
      unnamed:
        "Name this function as a component or hook: it calls {{hook}}, and the React " +
        "Compiler skips an unnamed function.",
    },
    schema: [],
  },
  create: (context) => {
    const hooks: (string | null)[] = [];
    const enter = (): void => {
      hooks.push(null);
    };
    const exit = (node: Rule.Node): void => {
      const hook = hooks.pop();
      if (hook == null || nameOf(node) != null) return;
      if (isForwardingMember(node) || insideComponentOrHook(node)) return;
      context.report({ node, messageId: "unnamed", data: { hook } });
    };
    return {
      ArrowFunctionExpression: enter,
      FunctionExpression: enter,
      FunctionDeclaration: enter,
      "ArrowFunctionExpression:exit": exit,
      "FunctionExpression:exit": exit,
      "FunctionDeclaration:exit": exit,
      CallExpression: (node) => {
        const name = calleeName(node.callee as Rule.Node);
        if (name == null || !HOOK.test(name) || hooks.length === 0) return;
        hooks[hooks.length - 1] ??= name;
      },
    };
  },
};
