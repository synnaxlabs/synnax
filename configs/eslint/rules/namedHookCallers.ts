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
const NAMED = /^(use([A-Z0-9]|$)|[A-Z])/;
const NAMESPACE = /^[A-Z]/;
const WRAPPERS = new Set(["memo", "forwardRef"]);
// The casts that ../compiler.ts removes before the compiler runs.
const CASTS = new Set(["TSAsExpression", "TSSatisfiesExpression"]);
const FUNCTIONS = new Set([
  "ArrowFunctionExpression",
  "FunctionExpression",
  "FunctionDeclaration",
]);

/** @returns the hook a call invokes, by React's test: `useX()` or `Space.useX()`. */
const hookName = (callee: Rule.Node): string | null => {
  if (callee.type === "Identifier") return HOOK.test(callee.name) ? callee.name : null;
  if (
    callee.type === "MemberExpression" &&
    !callee.computed &&
    callee.object.type === "Identifier" &&
    NAMESPACE.test(callee.object.name) &&
    callee.property.type === "Identifier" &&
    HOOK.test(callee.property.name)
  )
    return callee.property.name;
  return null;
};

const uncast = (fn: Rule.Node): Rule.Node => {
  let node = fn;
  while (node.parent != null && CASTS.has(node.parent.type)) node = node.parent;
  return node;
};

/** The compiler compiles a memo or forwardRef callback as a component. */
const isWrapped = (fn: Rule.Node): boolean => {
  const { parent } = uncast(fn);
  if (parent?.type !== "CallExpression") return false;
  const { callee } = parent;
  if (callee.type === "Identifier") return WRAPPERS.has(callee.name);
  return (
    callee.type === "MemberExpression" &&
    callee.object.type === "Identifier" &&
    callee.object.name === "React" &&
    callee.property.type === "Identifier" &&
    WRAPPERS.has(callee.property.name)
  );
};

/**
 * @returns the name the compiler reads for the function: its declared name or the
 * variable it initializes. The compiler skips any other function.
 */
const nameOf = (fn: Rule.Node): string | null => {
  if (fn.type === "FunctionDeclaration") return fn.id?.name ?? null;
  const node = uncast(fn);
  const { parent } = node;
  if (
    parent?.type === "VariableDeclarator" &&
    parent.init === node &&
    parent.id.type === "Identifier"
  )
    return parent.id.name;
  return null;
};

/** A `useX: () => useY(...)` member only forwards a hook, so it has nothing to memo. */
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
    if (FUNCTIONS.has(node.type))
      return isWrapped(node) || NAMED.test(nameOf(node) ?? "");
  return false;
};

export const namedHookCallers: Rule.RuleModule = {
  meta: {
    type: "problem",
    docs: {
      description:
        "Require a function that calls a hook to be declared or bound to a variable, " +
        "so the React Compiler compiles it.",
    },
    messages: {
      unnamed:
        "Declare this function or bind it to a variable with a component or hook " +
        "name: it calls {{hook}}, and the React Compiler skips a function it cannot " +
        "name.",
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
      if (hook == null || nameOf(node) != null || isWrapped(node)) return;
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
        const hook = hookName(node.callee as Rule.Node);
        if (hook == null || hooks.length === 0) return;
        hooks[hooks.length - 1] ??= hook;
      },
    };
  },
};
