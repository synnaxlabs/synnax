// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import fs from "fs";
import os from "os";
import path from "path";
import postcssHtml from "postcss-html";
import stylelint from "stylelint";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { type Baseline, plugins } from "./rules.ts";
import { read } from "./source.ts";

describe("plugins", () => {
  let root: string;
  beforeEach(() => {
    root = fs.mkdtempSync(path.join(os.tmpdir(), "stylelint-rules-"));
    const files: Record<string, string> = {
      "lyra/src/button/Button.tsx": `import { CSS } from "@/css"; CSS.B("btn");`,
      "lyra/src/theming/base.css": `:root { --pluto-gray-l8: #000; }`,
      "console/src/nav/Nav.tsx": `import { CSS } from "@/platform/css"; CSS.B("nav");`,
    };
    for (const [file, text] of Object.entries(files)) {
      fs.mkdirSync(path.dirname(path.join(root, file)), { recursive: true });
      fs.writeFileSync(path.join(root, file), text);
    }
  });
  afterEach(() => fs.rmSync(root, { recursive: true, force: true }));

  const lint = async (
    file: string,
    code: string,
    rule: string,
    baseline: Partial<Baseline> = {},
  ): Promise<string[]> => {
    const full = path.join(root, file);
    fs.mkdirSync(path.dirname(full), { recursive: true });
    fs.writeFileSync(full, code);
    const { results } = await stylelint.lint({
      code,
      codeFilename: path.join(root, file),
      config: {
        customSyntax: file.endsWith(".astro") ? postcssHtml : undefined,
        plugins: plugins({
          root,
          source: read(root),
          baseline: { foreignClasses: {}, globalStyles: [], ...baseline },
        }),
        rules: { [rule]: true },
      },
    });
    return results[0].warnings.map((w) => w.text);
  };

  it("should reject a baseline that lists a file that does not exist", () => {
    const baseline = {
      foreignClasses: { "console/src/Gone.css": ["pluto-btn"] },
      globalStyles: [],
    };
    expect(() => plugins({ root, source: read(root), baseline })).toThrow(
      "configs/stylelint/baseline.json lists files that do not exist: " +
        "console/src/Gone.css. Delete their entries.",
    );
  });

  describe("synnax/no-unknown-class", () => {
    const RULE = "synnax/no-unknown-class";

    it("should report a class that no code emits", async () => {
      expect(await lint("lyra/src/a.css", ".lyra-btnn { color: red; }", RULE)).toEqual([
        `Unexpected class "lyra-btnn" that no code emits (${RULE})`,
      ]);
    });

    it("should accept an emitted class and a class outside the BEM prefixes", async () => {
      const code = ".lyra-btn .header { color: red; }";
      expect(await lint("lyra/src/a.css", code, RULE)).toEqual([]);
    });
  });

  describe("synnax/no-unknown-custom-property", () => {
    const RULE = "synnax/no-unknown-custom-property";

    it("should report a fallback that nothing defines", async () => {
      const code = ".a { color: var(--pluto-gray-l8, var(--pluto-nope)); }";
      expect(await lint("lyra/src/a.css", code, RULE)).toEqual([
        `Unexpected custom property "--pluto-nope" that nothing defines (${RULE})`,
      ]);
    });

    it("should skip a var() inside a string", async () => {
      const code = `.a::after { content: "var(--pluto-nope)"; }`;
      expect(await lint("lyra/src/a.css", code, RULE)).toEqual([]);
    });
  });

  describe("synnax/no-foreign-class", () => {
    const RULE = "synnax/no-foreign-class";
    const FILE = "console/src/nav/Nav.css";

    it("should report a class that only another package emits", async () => {
      expect(await lint(FILE, ".console-nav .lyra-btn { color: red; }", RULE)).toEqual([
        `Unexpected class "lyra-btn" that lyra owns. Pass your own class through ` +
          `className, or set a custom property that lyra documents. (${RULE})`,
      ]);
    });

    it("should accept a class that the baseline lists for the file", async () => {
      const baseline = { foreignClasses: { [FILE]: ["lyra-btn"] } };
      expect(await lint(FILE, ".lyra-btn { color: red; }", RULE, baseline)).toEqual([]);
    });

    it("should report a baseline entry that the file no longer uses", async () => {
      const baseline = { foreignClasses: { [FILE]: ["lyra-btn"] } };
      expect(await lint(FILE, ".console-nav { color: red; }", RULE, baseline)).toEqual([
        `Unexpected baseline entry "lyra-btn" that this file no longer styles. Delete ` +
          `it from configs/stylelint/baseline.json. (${RULE})`,
      ]);
    });

    it("should judge an Astro file once across its style blocks", async () => {
      const file = "site/docs/src/Page.astro";
      const code = `<div style="--x: 1"></div>
<style>.lyra-btn { color: red; }</style>`;
      const baseline = { foreignClasses: { [file]: ["lyra-btn"] } };
      expect(await lint(file, code, RULE, baseline)).toEqual([]);
    });
  });

  describe("synnax/no-global-astro-style", () => {
    const RULE = "synnax/no-global-astro-style";
    const GLOBAL = "<div></div>\n<style is:global>.a { color: red; }</style>\n";

    it("should report a global style block in a component", async () => {
      expect(await lint("site/docs/src/components/A.astro", GLOBAL, RULE)).toEqual([
        "Unexpected global style block outside a layout. Write a scoped <style>, and " +
          `wrap a selector for a child component's markup in :global(). (${RULE})`,
      ]);
    });

    it("should accept a global style block in a layout", async () => {
      expect(await lint("site/docs/src/layouts/A.astro", GLOBAL, RULE)).toEqual([]);
    });

    it("should report a baseline entry for a file that scopes its styles", async () => {
      const file = "site/docs/src/components/A.astro";
      const code = "<div></div>\n<style>.a { color: red; }</style>\n";
      expect(await lint(file, code, RULE, { globalStyles: [file] })).toEqual([
        "Unexpected baseline entry for a file with no global style block. Delete it " +
          `from configs/stylelint/baseline.json. (${RULE})`,
      ]);
    });
  });
});
