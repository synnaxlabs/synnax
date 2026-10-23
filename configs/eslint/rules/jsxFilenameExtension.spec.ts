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

import { jsxFilenameExtension } from "./jsxFilenameExtension";

RuleTester.describe = describe;
RuleTester.it = it;
RuleTester.itOnly = it.only;

const tester = new RuleTester({ languageOptions: { parser: tseslint.parser } });

tester.run("jsx-filename-extension", jsxFilenameExtension, {
  valid: [
    { filename: "a.tsx", code: "export const A = () => <div />;" },
    { filename: "a.tsx", code: "export const A = () => <></>;" },
    { filename: "a.ts", code: "export const a = 1;" },
  ],
  invalid: [
    {
      filename: "a.tsx",
      code: "export const a = 1;",
      errors: [{ messageId: "noJSX", line: 1, column: 20 }],
    },
  ],
});
