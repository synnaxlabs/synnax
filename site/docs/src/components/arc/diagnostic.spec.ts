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
  it("should return the kinds by band, with warnings before errors", () => {
    const codes = Arc.kinds().map((kind) => kind.code);
    expect(codes.slice(0, 7)).toEqual([
      "E1000",
      "E1001",
      "E1002",
      "E1003",
      "W2000",
      "W2001",
      "W2002",
    ]);
    const from = codes.indexOf("E3013");
    expect(codes.slice(from, from + 4)).toEqual(["E3013", "W4000", "W4001", "E4000"]);
    expect(codes.at(-1)).toEqual("E5015");
  });

  it("should leave out the internal errors of band 0", () => {
    expect(Arc.kinds().filter(({ code }) => code[1] === "0")).toEqual([]);
  });

  it("should name the band of each kind", () => {
    for (const { code } of Arc.kinds()) expect(Arc.BANDS[code[1]]).toBeDefined();
  });

  it("should give a kind its severity, problem, example, and message", () => {
    const kind = Arc.kinds().find(({ code }) => code === "W4000");
    expect(kind).toEqual({
      code: "W4000",
      severity: "warning",
      message: 'imported module "math" is unused',
      problem: "The code imports a module and does not use it.",
      example: "import math",
    });
  });

  it("should give a kind with no example an empty message", () => {
    const kind = Arc.kinds().find(({ code }) => code === "E1001");
    expect(kind?.message).toEqual("");
    expect(kind?.example).toEqual("");
  });
});
