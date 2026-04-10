// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { fmt } from "@/fmt";

describe("fmt.path", () => {
  it("should render an empty path as <root>", () => {
    expect(fmt.path([])).toBe("<root>");
  });

  it("should render a single string segment without a leading dot", () => {
    expect(fmt.path(["name"])).toBe("name");
  });

  it("should join string segments with dots", () => {
    expect(fmt.path(["a", "b", "c"])).toBe("a.b.c");
  });

  it("should render a single numeric segment as a bracket index", () => {
    expect(fmt.path([0])).toBe("[0]");
  });

  it("should render numeric segments as bracket indices in a mixed path", () => {
    expect(fmt.path(["items", 0, "name"])).toBe("items[0].name");
  });

  it("should render consecutive numeric segments as adjacent brackets", () => {
    expect(fmt.path(["matrix", 1, 2])).toBe("matrix[1][2]");
  });

  it("should render a numeric root followed by a string as [0].name", () => {
    expect(fmt.path([0, "name"])).toBe("[0].name");
  });

  it("should render a symbol segment via its String() form", () => {
    expect(fmt.path([Symbol("foo")])).toBe("Symbol(foo)");
  });
});
