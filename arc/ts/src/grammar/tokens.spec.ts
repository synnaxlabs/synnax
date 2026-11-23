// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type Token, tokens } from "@/index";

const HEX_COLOR = /^#[0-9a-fA-F]{6}$/;

describe("tokens", () => {
  it("should give every token type a hex color for each theme", () => {
    Object.values(tokens).forEach((token: Token) => {
      expect(token.dark).toMatch(HEX_COLOR);
      expect(token.light).toMatch(HEX_COLOR);
    });
  });

  it("should assign each scope to one token type only", () => {
    const scopes = Object.values(tokens).flatMap((t) => t.scopes);
    expect(new Set(scopes).size).toBe(scopes.length);
  });

  it("should limit every specific scope to the Arc language", () => {
    const scopes = Object.values(tokens).flatMap((t) => t.scopes);
    const generic = new Set(["comment", "constant.numeric"]);
    scopes
      .filter((scope) => !generic.has(scope))
      .forEach((scope) => expect(scope).toMatch(/\.arc$/));
  });
});
