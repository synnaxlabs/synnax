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

import { aliasImports } from "./aliasImports";

RuleTester.describe = describe;
RuleTester.it = it;
RuleTester.itOnly = it.only;

const root = fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(), "alias-imports-")));
const write = (file: string, content = ""): void => {
  const full = path.join(root, file);
  fs.mkdirSync(path.dirname(full), { recursive: true });
  fs.writeFileSync(full, content);
};
write("aliased/package.json");
write(
  "aliased/tsconfig.json",
  '{ "compilerOptions": { "paths": { "@/*": ["./src/*"] } } }',
);
write("plain/package.json");
write("plain/tsconfig.json", "{}");

afterAll(() => fs.rmSync(root, { recursive: true, force: true }));

const file = (name: string): string => path.join(root, name);

const tester = new RuleTester({ languageOptions: { parser: tseslint.parser } });

const relative = (
  code: string,
  output: string,
  alias: string,
): RuleTester.InvalidTestCase => ({
  filename: file("aliased/src/a/b.ts"),
  code,
  output,
  errors: [{ messageId: "relative", data: { alias } }],
});

tester.run("alias-imports", aliasImports, {
  valid: [
    { filename: file("aliased/src/a/b.ts"), code: 'import { c } from "@/a/c";' },
    { filename: file("aliased/src/a/b.ts"), code: 'import { useState } from "react";' },
    {
      filename: file("aliased/src/a/b.ts"),
      code: 'import config from "../../vite.config";',
    },
    { filename: file("aliased/src/a/b.ts"), code: 'import { c } from "./c.ts";' },
    {
      filename: file("aliased/vite.config.ts"),
      code: 'import { c } from "./src/a/c";',
    },
    { filename: file("plain/src/a/b.ts"), code: 'import { c } from "./c";' },
  ],
  invalid: [
    relative('import { c } from "./c";', 'import { c } from "@/a/c";', "@/a/c"),
    relative('import { d } from "../d/d";', 'import { d } from "@/d/d";', "@/d/d"),
    relative('import "./b.css";', 'import "@/a/b.css";', "@/a/b.css"),
    relative('export * from "./c";', 'export * from "@/a/c";', "@/a/c"),
    relative('export { c } from "./c";', 'export { c } from "@/a/c";', "@/a/c"),
    relative('void import("./c");', 'void import("@/a/c");', "@/a/c"),
  ],
});
