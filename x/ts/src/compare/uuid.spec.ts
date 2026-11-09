// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { readFileSync } from "node:fs";

import { describe, expect, it } from "vitest";

import { compare } from "@/compare";

const VECTORS_URL = new URL("../../../testdata/compare/uuid.json", import.meta.url);

const vectors = JSON.parse(readFileSync(VECTORS_URL, "utf-8")) as { ordered: string[] };

describe("uuids", () => {
  it("should order each vector before the next", () => {
    for (let i = 1; i < vectors.ordered.length; i++) {
      const a = vectors.ordered[i - 1];
      const b = vectors.ordered[i];
      expect(compare.uuids(a, b), `${a} before ${b}`).toBe(-1);
      expect(compare.uuids(b, a), `${b} after ${a}`).toBe(1);
    }
  });

  it("should order a UUID the same as itself", () => {
    expect(compare.uuids(vectors.ordered[0], vectors.ordered[0])).toBe(0);
  });
});
