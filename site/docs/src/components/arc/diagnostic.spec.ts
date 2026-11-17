// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Arc } from "@/components/arc";

describe("kinds", () => {
  it("should return the kinds of a band with errors before warnings", () => {
    const codes = Arc.kinds(4).map((kind) => kind.code);
    expect(codes).toEqual([
      "E4000",
      "E4001",
      "E4002",
      "E4003",
      "E4004",
      "E4005",
      "E4006",
      "E4007",
      "E4008",
      "E4009",
      "W4000",
      "W4001",
    ]);
  });

  it("should give a kind its severity, problem, example, and message", () => {
    const kind = Arc.kinds(4).find(({ code }) => code === "W4000");
    expect(kind).toEqual({
      code: "W4000",
      severity: "warning",
      message: 'imported module "math" is unused',
      problem: "The code imports a module and does not use it.",
      example: "import math",
    });
  });

  it("should give a kind with no example an empty message", () => {
    const kind = Arc.kinds(1).find(({ code }) => code === "E1001");
    expect(kind?.message).toEqual("");
    expect(kind?.example).toEqual("");
  });

  it("should throw for a band that has no kind", () => {
    expect(() => Arc.kinds(9)).toThrow("no Arc diagnostic kind is in band 9");
  });
});
