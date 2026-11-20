// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { grammar, grammarRaw, languageConfigurationRaw, tokens } from "@/index";

const collectScopes = (value: unknown, scopes: Set<string>): Set<string> => {
  if (Array.isArray(value)) value.forEach((v) => collectScopes(v, scopes));
  else if (typeof value === "object" && value !== null)
    Object.entries(value).forEach(([key, v]) => {
      if (key === "name" && typeof v === "string") scopes.add(v);
      else collectScopes(v, scopes);
    });
  return scopes;
};

describe("grammar", () => {
  it("should name the language and its root scope", () => {
    expect(grammar.name).toBe("arc");
    expect(grammar.scopeName).toBe("source.arc");
  });

  it("should hold the patterns and repository of the raw grammar", () => {
    const raw: unknown = JSON.parse(grammarRaw);
    expect(raw).toMatchObject({
      patterns: grammar.patterns,
      repository: grammar.repository,
    });
    expect(grammar.patterns.length).toBeGreaterThan(0);
  });

  it("should resolve every include to a repository entry", () => {
    const includes = grammarRaw.matchAll(/"include":\s*"#([\w-]+)"/g);
    for (const [, key] of includes) expect(grammar.repository).toHaveProperty(key);
  });

  it("should color every scope it emits except meta scopes", () => {
    const tokenScopes = Object.values(tokens).flatMap((t) => t.scopes);
    const scopes = collectScopes(grammar.repository, new Set());
    const uncolored = [...scopes].filter(
      (scope) =>
        !scope.startsWith("meta.") &&
        !tokenScopes.some((t) => scope === t || scope.startsWith(`${t}.`)),
    );
    expect(uncolored).toEqual([]);
  });
});

describe("languageConfigurationRaw", () => {
  it("should use C-style comments", () => {
    expect(JSON.parse(languageConfigurationRaw)).toMatchObject({
      comments: { lineComment: "//", blockComment: ["/*", "*/"] },
    });
  });

  it("should auto-close every bracket", () => {
    const config: { brackets: string[][]; autoClosingPairs: string[][] } = JSON.parse(
      languageConfigurationRaw,
    );
    expect(config.autoClosingPairs).toEqual(expect.arrayContaining(config.brackets));
  });
});
