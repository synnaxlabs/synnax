// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Builds compile with these passes too (configs/vite). Lint cannot depend on a built
// package, so they live here.

import { existsSync, readFileSync } from "node:fs";
import path from "node:path";

import {
  type NodePath,
  type PluginItem,
  type PluginObj,
  types as t,
  type Visitor,
} from "@babel/core";

const PLUTO = "@synnaxlabs/pluto";
const NAMESPACE = /^export \* as (\w+) from /m;
const LOWERCASE = /^[a-z]/;

const read = (file: string): string =>
  existsSync(file) ? readFileSync(file, "utf8") : "";

/**
 * Reports whether a file is Aether worker code, which names its functions like hooks
 * but runs outside React. Worker code is every lowercase namespace in Pluto, read from
 * the nearest `index.ts` above the file.
 */
export const isWorker = (file: string): boolean => {
  let namespace: string | undefined;
  for (let dir = path.dirname(file); ; dir = path.dirname(dir)) {
    namespace ??= NAMESPACE.exec(read(path.join(dir, "index.ts")))?.[1];
    if (namespace != null && !LOWERCASE.test(namespace)) return false;
    const manifest = read(path.join(dir, "package.json"));
    if (manifest !== "")
      return namespace != null && JSON.parse(manifest).name === PLUTO;
    if (path.dirname(dir) === dir) return false;
  }
};

const SENTINEL = /^react\./;

/**
 * Points each compiler sentinel at the global `Symbol` in a file that binds its own
 * `Symbol`, such as a component named `Symbol`. The compiler emits a bare `Symbol.for`.
 */
const GLOBAL_SYMBOL: PluginObj = {
  visitor: {
    Program: {
      exit: (program) =>
        program.traverse({
          CallExpression: (call) => {
            const {
              callee,
              arguments: [key],
            } = call.node;
            if (
              !t.isMemberExpression(callee) ||
              !t.isIdentifier(callee.object, { name: "Symbol" }) ||
              !t.isIdentifier(callee.property, { name: "for" }) ||
              !t.isStringLiteral(key) ||
              !SENTINEL.test(key.value) ||
              !call.scope.hasBinding("Symbol", true)
            )
              return;
            callee.object = t.memberExpression(
              t.identifier("globalThis"),
              t.identifier("Symbol"),
            );
          },
        }),
    },
  },
};

/**
 * Removes a TypeScript cast around a function, which hides a component or hook from the
 * compiler. `tsc` output has no casts, so source and emitted builds compile alike.
 */
const UNWRAP_CASTS: PluginObj = {
  visitor: {
    Program: (program) =>
      program.traverse({
        "TSAsExpression|TSSatisfiesExpression": (cast: NodePath) => {
          const inner = cast.get("expression") as NodePath;
          if (inner.isFunction()) cast.replaceWith(inner.node);
        },
      }),
  },
};

// The compiler only treats `use` followed by a capital letter or a digit as a hook
// name, apart from React's own `use`.
const BARE = "use";
const MARKED = "useBare$";

type Binding = ReturnType<NodePath["scope"]["getBinding"]>;

const isReact = (binding: Binding): boolean => {
  const declaration = binding?.path.parentPath;
  return (
    declaration?.isImportDeclaration() === true &&
    declaration.node.source.value === "react"
  );
};

const renameUse = (
  program: NodePath<t.Program>,
  from: string,
  to: string,
  visitor: Visitor = {},
): void => {
  const rename = (scoped: NodePath) => {
    const binding = scoped.scope.getOwnBinding(from);
    if (binding != null && !isReact(binding)) scoped.scope.rename(from, to);
  };
  const renameProperty = <M extends t.MemberExpression | t.OptionalMemberExpression>(
    member: NodePath<M>,
  ) => {
    const { object, property, computed } = member.node;
    if (computed || property.type !== "Identifier" || property.name !== from) return;
    if (object.type === "Identifier" && isReact(member.scope.getBinding(object.name)))
      return;
    property.name = to;
  };
  // The compiler types a destructured binding by its key, not by its own name.
  const renameKey = (property: NodePath<t.ObjectProperty>) => {
    const { key, value, computed } = property.node;
    if (computed || key.type !== "Identifier" || key.name !== from) return;
    if (!property.parentPath.isObjectPattern()) return;
    key.name = to;
    property.node.shorthand = value.type === "Identifier" && value.name === to;
  };
  rename(program);
  program.traverse({
    ...visitor,
    Scopable: rename,
    MemberExpression: renameProperty,
    OptionalMemberExpression: renameProperty,
    ObjectProperty: renameKey,
  });
};

const rejectMarked = (reason: string): Visitor => ({
  Identifier: (id) => {
    if (id.node.name === MARKED) throw id.buildCodeFrameError(`${MARKED} ${reason}`);
  },
});

/**
 * Gives each `use` variable and `.use` property that is not React's `use` a hook name,
 * so the compiler compiles and calls them as hooks.
 */
const MARK_BARE_USE: PluginObj = {
  visitor: {
    Program: (program, { file }) => {
      if (file.code.includes(MARKED))
        program.traverse(rejectMarked("is reserved for the React Compiler"));
      renameUse(program, BARE, MARKED);
    },
  },
};

const HOOK = /^use[A-Z0-9]/;
const DIRECTIVES = new Set(["use memo", "use no memo", "use forget", "use no forget"]);

const functionName = (fn: NodePath<t.Function>): string | null => {
  if (fn.isFunctionDeclaration()) return fn.node.id?.name ?? null;
  const { parent } = fn;
  let id: t.Node | null = null;
  if (t.isVariableDeclarator(parent) && parent.init === fn.node) id = parent.id;
  else if (t.isAssignmentExpression(parent) && parent.right === fn.node)
    id = parent.left;
  else if (t.isObjectProperty(parent) && parent.value === fn.node && !parent.computed)
    id = parent.key;
  if (t.isMemberExpression(id) && !id.computed) id = id.property;
  return t.isIdentifier(id) ? id.name : null;
};

const hasDirective = (directives: t.Directive[]): boolean =>
  directives.some((d) => DIRECTIVES.has(d.value.value));

const MEMO_APIS = new Set(["memo", "forwardRef"]);

const isMemoCallback = (fn: NodePath<t.Function>): boolean => {
  const { parent } = fn;
  if (!t.isCallExpression(parent) || parent.arguments[0] !== fn.node) return false;
  const { callee } = parent;
  const api =
    t.isMemberExpression(callee) && t.isIdentifier(callee.object, { name: "React" })
      ? callee.property
      : callee;
  return t.isIdentifier(api) && MEMO_APIS.has(api.name);
};

/**
 * Compiles every function named like a hook, as rules-of-hooks treats it, and every
 * `memo` or `forwardRef` component. The compiler skips one unless it sees a hook call
 * or JSX in it, and it misses hook calls through a nested or lowercase namespace, such
 * as `Session.Panel.useGetSelected()`.
 */
const COMPILE_HOOKS_AND_MEMOS: PluginObj = {
  visitor: {
    Program: (program, { filename }) => {
      if (filename != null && isWorker(filename)) return;
      if (hasDirective(program.node.directives)) return;
      program.traverse({
        Function: (fn) => {
          const name = functionName(fn);
          if ((name == null || !HOOK.test(name)) && !isMemoCallback(fn)) return;
          fn.skip();
          const directive = t.directive(t.directiveLiteral("use memo"));
          const { body } = fn.node;
          if (!t.isBlockStatement(body))
            fn.node.body = t.blockStatement([t.returnStatement(body)], [directive]);
          else if (!hasDirective(body.directives)) body.directives.unshift(directive);
        },
      });
    },
  },
};

/** Reverses {@link MARK_BARE_USE} after the compiler runs. */
const RESTORE_BARE_USE: PluginObj = {
  visitor: {
    Program: {
      // A scope renames its bindings before the traversal reaches their names.
      exit: (program) =>
        renameUse(program, MARKED, BARE, rejectMarked("is left after the compiler")),
    },
  },
};

/**
 * @returns the Babel plugins that run the React Compiler on Synnax code: the given
 * compiler plugin, with the passes it needs before and after it.
 * @throws during a transform of a file that contains the reserved name `useBare$`.
 */
export const plugins = (compiler: PluginItem): PluginItem[] => [
  UNWRAP_CASTS,
  MARK_BARE_USE,
  COMPILE_HOOKS_AND_MEMOS,
  compiler,
  RESTORE_BARE_USE,
  GLOBAL_SYMBOL,
];
