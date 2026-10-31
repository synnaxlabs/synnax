// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { RuleTester } from "eslint";
import fs from "fs";
import os from "os";
import path from "path";
import tseslint from "typescript-eslint";
import { afterAll, describe, it } from "vitest";

import { barrelImports } from "./barrelImports";

RuleTester.describe = describe;
RuleTester.it = it;
RuleTester.itOnly = it.only;

const root = fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(), "barrel-imports-")));
[
  "package.json",
  "src/a/index.ts",
  "src/a/external.ts",
  "src/a/Button.tsx",
  "src/a/testutil.ts",
  "src/a/plain/y.ts",
  "src/a/child/index.ts",
  "src/a/child/x.ts",
  "src/b/index.ts",
  "src/b/z.ts",
  "src/b/external.ts",
  "src/loose/w.ts",
  "src/pages/api/index.ts",
  "src/pages/api/route.ts",
].forEach((file) => {
  const full = path.join(root, file);
  fs.mkdirSync(path.dirname(full), { recursive: true });
  fs.writeFileSync(full, "");
});

afterAll(() => fs.rmSync(root, { recursive: true, force: true }));

const file = (name: string): string => path.join(root, name);

const tester = new RuleTester({ languageOptions: { parser: tseslint.parser } });

tester.run("barrel-imports", barrelImports, {
  valid: [
    { filename: file("src/b/z.ts"), code: 'import { A } from "@/a";' },
    { filename: file("src/b/z.ts"), code: 'import { A } from "../a";' },
    { filename: file("src/b/z.ts"), code: 'import { Child } from "@/a/child";' },
    { filename: file("src/b/z.ts"), code: 'import { useState } from "react";' },
    { filename: file("src/b/z.ts"), code: 'import { GET } from "@/pages/api/route";' },
    { filename: file("src/b/z.ts"), code: 'import "@/a/Button.css";' },
    { filename: file("src/a/Button.tsx"), code: 'export * from "@/a/external";' },
    { filename: file("src/a/Button.tsx"), code: 'import { y } from "./plain/y";' },
    { filename: file("src/a/child/x.ts"), code: 'import { B } from "@/a/Button";' },
    { filename: file("src/b/z.spec.ts"), code: 'import { B } from "@/b";' },
    {
      filename: file("src/b/z.spec.ts"),
      code: 'import { render } from "@/a/testutil";',
    },
    { filename: file("src/b/external.ts"), code: 'export * from "@/a/external";' },
    { filename: file("src/loose/w.spec.ts"), code: 'import { w } from "./w";' },
    { filename: file("vite.config.ts"), code: 'import { B } from "./src/a/Button";' },
  ],
  invalid: [
    {
      filename: file("src/b/external.ts"),
      code: 'export * as A from "@/a/external";',
      errors: [{ messageId: "deep", data: { module: "@/a" } }],
    },
    {
      filename: file("src/b/external.ts"),
      code: 'export { a } from "@/a/external";',
      errors: [{ messageId: "deep", data: { module: "@/a" } }],
    },
    {
      filename: file("src/b/z.ts"),
      code: 'export * from "@/a/external";',
      errors: [{ messageId: "deep", data: { module: "@/a" } }],
    },
    {
      filename: file("src/b/z.ts"),
      code: 'import { B } from "@/a/Button";',
      errors: [{ messageId: "deep", data: { module: "@/a" } }],
    },
    {
      filename: file("src/b/z.ts"),
      code: 'import { B } from "@/a/external";',
      errors: [{ messageId: "deep", data: { module: "@/a" } }],
    },
    {
      filename: file("src/b/z.ts"),
      code: 'import { y } from "../a/plain/y";',
      errors: [{ messageId: "deep", data: { module: "@/a" } }],
    },
    {
      filename: file("src/b/z.ts"),
      code: 'export * from "@/a/Button";',
      errors: [{ messageId: "deep", data: { module: "@/a" } }],
    },
    {
      filename: file("src/b/z.ts"),
      code: 'const m = await import("@/a/Button");',
      errors: [{ messageId: "deep", data: { module: "@/a" } }],
    },
    {
      filename: file("src/a/Button.tsx"),
      code: 'import { x } from "@/a/child/x";',
      errors: [{ messageId: "deep", data: { module: "@/a/child" } }],
    },
    {
      filename: file("src/a/Button.spec.tsx"),
      code: 'import { B } from "./Button";',
      errors: [{ messageId: "spec", data: { file: "Button.tsx" } }],
    },
    {
      filename: file("src/a/child/x.spec.ts"),
      code: 'import { B } from "@/a/Button";',
      errors: [{ messageId: "spec", data: { file: "../Button.tsx" } }],
    },
  ],
});
