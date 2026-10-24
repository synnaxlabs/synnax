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

import { reactCompiler } from "./reactCompiler";

RuleTester.describe = describe;
RuleTester.it = it;
RuleTester.itOnly = it.only;

const tester = new RuleTester({ languageOptions: { parser: tseslint.parser } });

const REF_IN_RENDER = [
  'import { useRef } from "react";',
  "export const use = () => {",
  "  const r = useRef(0);",
  "  return r.current;",
  "};",
].join("\n");

const CONDITIONAL_HOOK = [
  'import { Synnax } from "@synnaxlabs/pluto";',
  "export const C = ({ on }: { on: boolean }) => {",
  "  const c = on ? Synnax.use() : null;",
  "  return <p>{c?.k}</p>;",
  "};",
].join("\n");

tester.run("react-compiler", reactCompiler, {
  valid: [
    {
      code: 'import { Synnax } from "@synnaxlabs/pluto";\nexport const use = () => Synnax.use();',
      filename: "file.ts",
      options: [{ rules: ["hooks", "refs"] }],
    },
    { code: REF_IN_RENDER, filename: "file.ts", options: [{ rules: ["hooks"] }] },
  ],
  invalid: [
    {
      code: REF_IN_RENDER,
      filename: "file.ts",
      options: [{ rules: ["refs"] }],
      errors: [{ message: /^Error: Cannot access refs during render\n/, line: 4 }],
    },
    {
      code: CONDITIONAL_HOOK,
      filename: "file.tsx",
      options: [{ rules: ["hooks"] }],
      errors: [
        {
          message: /^Error: Hooks must always be called in a consistent order/,
          line: 3,
          column: 18,
        },
      ],
    },
  ],
});
