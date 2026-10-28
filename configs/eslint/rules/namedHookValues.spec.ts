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

import { namedHookValues } from "./namedHookValues";

RuleTester.describe = describe;
RuleTester.it = it;
RuleTester.itOnly = it.only;

const tester = new RuleTester({ languageOptions: { parser: tseslint.parser } });

const renamed = (hook: string, name: string, column: number) => ({
  messageId: "renamed",
  data: { hook, name },
  line: 1,
  column,
});

tester.run("named-hook-values", namedHookValues, {
  valid: [
    { code: "const useKey = Scope.use;" },
    { code: "const use = Synnax.use;" },
    { code: "const useA = useState;" },
    { code: "const x = { use: Scope.use, useKey: Scope.useKey };" },
    { code: "const { use } = Synnax;" },
    { code: "const { use: useClient } = Synnax;" },
    { code: 'import { use } from "./hooks";' },
    { code: 'import { useA as useB } from "./hooks";' },
    { code: "export { use as useClient };" },
    { code: "const c = Synnax.use();" },
    { code: "const user = Synnax.user;" },
    { code: "f(Scope.use);" },
    { code: "const { [k]: get } = Synnax;" },
  ],
  invalid: [
    {
      code: "const get = Scope.use;",
      errors: [renamed("use", "get", 13)],
    },
    {
      code: "const state = useState;",
      errors: [renamed("useState", "state", 15)],
    },
    {
      code: "const x = { get: Synnax.use };",
      errors: [renamed("use", "get", 18)],
    },
    {
      code: "const { use: client } = Synnax;",
      errors: [renamed("use", "client", 9)],
    },
    {
      code: "const { use: client = f } = Synnax;",
      errors: [renamed("use", "client", 9)],
    },
    {
      code: 'import { useA as a } from "./hooks";',
      errors: [renamed("useA", "a", 10)],
    },
    {
      code: "export { use as client };",
      errors: [renamed("use", "client", 10)],
    },
    {
      code: "let get; get = Scope.use;",
      errors: [renamed("use", "get", 16)],
    },
  ],
});
