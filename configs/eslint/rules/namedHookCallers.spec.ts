// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { RuleTester } from "eslint";
import tseslint from "typescript-eslint";
import { describe, it } from "vitest";

import { namedHookCallers } from "./namedHookCallers";

RuleTester.describe = describe;
RuleTester.it = it;
RuleTester.itOnly = it.only;

const tester = new RuleTester({ languageOptions: { parser: tseslint.parser } });

const unnamed = (hook: string, line: number, column: number) => ({
  messageId: "unnamed",
  data: { hook },
  line,
  column,
});

tester.run("named-hook-callers", namedHookCallers, {
  valid: [
    { code: "const useA = () => useState(0);" },
    { code: "function useA() { return useState(0); }" },
    { code: "const A = () => { useState(0); return null; };" },
    { code: "const A = memo(() => { useState(0); return null; });" },
    { code: "const A = forwardRef((p, r) => { useState(0); return null; });" },
    { code: "const A = React.memo(() => { useState(0); return null; });" },
    { code: "export default memo(() => { useState(0); return null; });" },
    { code: "const A = function B() { useState(0); return null; };" },
    { code: "const useA = (() => use(p)) as UseA;" },
    { code: "const create = () => { const useA = () => useState(0); return useA; };" },
    { code: "const x = { useVisible: () => Access.useGranted(id) };" },
    { code: "const x = { use: (q) => useSuspended(c, q) };" },
    { code: "const use = () => items.map((i) => Access.useGranted(i));" },
    { code: "const useA = () => items.map((i) => i.useVisible());" },
    { code: "const a = () => items.map((i) => i + 1);" },
    { code: "class Client { use(m) { this.unary.use(m); } }" },
  ],
  invalid: [
    {
      code: "const create = () => () => useState(0);",
      errors: [unnamed("useState", 1, 22)],
    },
    {
      code: "const item = renderProp((p) => { useItem(p.key); return null; });",
      errors: [unnamed("useItem", 1, 25)],
    },
    {
      code: "const sel = bindHook(({ key }) => Select.useMemo(f, [key]));",
      errors: [unnamed("useMemo", 1, 22)],
    },
    {
      code: "const x = { useVisible: () => { const a = useA(); return a; } };",
      errors: [unnamed("useA", 1, 25)],
    },
    {
      code: "const create = () => () => use(promise);",
      errors: [unnamed("use", 1, 22)],
    },
    {
      code: "const i = renderProp(function Item(p) { useItem(p.key); return null; });",
      errors: [unnamed("useItem", 1, 22)],
    },
    {
      code: "let useA; useA = () => useState(0);",
      errors: [unnamed("useState", 1, 18)],
    },
    {
      code: "const x = { useVisible() { return useA(); } };",
      errors: [unnamed("useA", 1, 23)],
    },
    {
      code: "const useA = (() => useState(0))!;",
      errors: [unnamed("useState", 1, 15)],
    },
    {
      code: "const A = Foo.memo(() => { useState(0); return null; });",
      errors: [unnamed("useState", 1, 20)],
    },
  ],
});
