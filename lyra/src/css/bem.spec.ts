// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, test } from "vitest";

import { newBEM } from "@/css/bem";

describe("BEM", () => {
  const bem = newBEM("test");
  test("B", () => expect(bem.B("a")).toBe("test-a"));
  test("E", () => expect(bem.E("a")).toBe("test__a"));
  test("M", () => expect(bem.M("a")).toBe("test--a"));
  test("BE", () => expect(bem.BE("a", "b")).toBe("test-a__b"));
  test("BM", () => expect(bem.BM("a", "b")).toBe("test-a--b"));
  test("BEM", () => expect(bem.BEM("a", "b", "c")).toBe("test-a__b--c"));
});
