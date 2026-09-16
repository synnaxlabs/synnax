// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import fs from "node:fs";
import path from "node:path";

import { describe, expect, it } from "vitest";

const SRC = path.resolve(import.meta.dirname, "..");
const TOKENS_FILE = "styles/base.css";
const LAYOUT_FILE = "styles/layout.css";
const TOKEN = /--docs-block-[a-z]+(?=:)/g;
const MEDIA_SELECTOR = "img:not(.diagram img, .blog-author__photo, figure img)";

// Every block that shares the docs geometry, with the rule that must consume it.
const CONSUMERS = [
  {
    name: "note",
    file: "components/article/Article.astro",
    selector: ".pluto-note",
    tokens: ["--docs-block-inset", "--docs-block-padding", "--docs-block-gap"],
  },
  {
    name: "code block",
    file: "components/code/Block.astro",
    selector: ".astro-code-wrapper",
    tokens: ["--docs-block-inset", "--docs-block-padding", "--docs-block-gap"],
  },
  {
    name: "media",
    file: LAYOUT_FILE,
    selector: MEDIA_SELECTOR,
    tokens: ["--docs-block-inset", "--docs-block-gap"],
  },
  {
    name: "diagram",
    file: "components/diagram/Diagram.astro",
    selector: ".diagram",
    tokens: ["--docs-block-inset", "--docs-block-gap"],
  },
];

const read = (rel: string): string => fs.readFileSync(path.join(SRC, rel), "utf8");

const escape = (s: string): string => s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

// Returns the rule's text from its selector through the matching close brace. The
// selector must open its line, so a longer compound selector cannot match.
const ruleBlock = (css: string, selector: string): string => {
  const start = css.search(new RegExp(`^\\s*${escape(selector)} \\{`, "m"));
  if (start === -1) throw new Error(`no rule for ${selector}`);
  let depth = 0;
  for (let i = css.indexOf("{", start); i < css.length; i++)
    if (css[i] === "{") depth += 1;
    else if (css[i] === "}" && --depth === 0) return css.slice(start, i + 1);
  throw new Error(`unterminated rule for ${selector}`);
};

const styleFiles = (): string[] =>
  fs
    .readdirSync(SRC, { recursive: true, encoding: "utf8" })
    .filter((f) => /\.(css|astro|tsx)$/.test(f));

describe("docs block tokens", () => {
  it("should declare every token in base.css", () => {
    const declared = new Set(read(TOKENS_FILE).match(TOKEN));
    const expected = new Set(CONSUMERS.flatMap((c) => c.tokens));
    expect(declared).toEqual(expected);
  });

  it("should drop the inset on narrow screens", () => {
    expect(read(TOKENS_FILE)).toMatch(
      /@media \(width <= 800px\) \{\s*--docs-block-inset: 0;/,
    );
  });

  it("should declare the tokens nowhere else", () => {
    const others = styleFiles()
      .filter((f) => f !== TOKENS_FILE)
      .filter((f) => read(f).search(TOKEN) !== -1);
    expect(others).toEqual([]);
  });

  CONSUMERS.forEach(({ name, file, selector, tokens }) => {
    it(`should size the ${name} with the tokens`, () => {
      const rule = ruleBlock(read(file), selector);
      for (const token of tokens) expect(rule).toContain(`var(${token})`);
    });
  });
});

describe("tab panels", () => {
  it("should lay out in block flow so margins collapse", () => {
    const rule = ruleBlock(read(LAYOUT_FILE), ".pluto-tabs__content");
    expect(rule).toContain("display: block");
  });
});
