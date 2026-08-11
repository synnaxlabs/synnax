// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { deduplicate } from "@/filename/deduplicate";

describe("deduplicate", () => {
  it("should return the original name when it does not exist", () =>
    expect(deduplicate("Report", new Set(["Summary"]))).toBe("Report"));

  it("should append (1) when a duplicate exists", () =>
    expect(deduplicate("Report", new Set(["Report"]))).toBe("Report (1)"));

  it("should increment to the next available number", () =>
    expect(deduplicate("Report", new Set(["Report", "Report (1)", "Report (2)"]))).toBe(
      "Report (3)",
    ));

  it("should fill gaps in numbering", () =>
    expect(deduplicate("Report", new Set(["Report", "Report (1)", "Report (3)"]))).toBe(
      "Report (2)",
    ));

  it("should handle names already suffixed with a number", () =>
    expect(deduplicate("Report (2)", new Set(["Report (2)"]))).toBe("Report (3)"));

  it("should not be confused by numbers elsewhere in the name", () =>
    expect(
      deduplicate("Report 2024", new Set(["Report 2024", "Report 2024 (1)"])),
    ).toBe("Report 2024 (2)"));

  it("should handle names with non-numeric parentheses at the end", () =>
    expect(deduplicate("Report (draft)", new Set(["Report (draft)"]))).toBe(
      "Report (draft) (1)",
    ));

  it("should escalate correctly when many duplicates exist", () => {
    const existing = new Set<string>(["Report"]);
    for (let i = 1; i <= 100; i++) existing.add(`Report (${i})`);
    expect(deduplicate("Report", existing)).toBe("Report (101)");
  });

  it("should return empty when empty name does not exist", () =>
    expect(deduplicate("", new Set(["Report"]))).toBe(""));

  it("should append (1) to empty when empty name exists", () =>
    expect(deduplicate("", new Set([""]))).toBe(" (1)"));

  // Edge cases
  it("should keep unique names with trailing spaces unchanged", () =>
    expect(deduplicate("Report ", new Set(["Report"]))).toBe("Report "));

  it("should normalize double spaces before numeric suffix when incrementing", () =>
    expect(deduplicate("Report  (1)", new Set(["Report  (1)"]))).toBe("Report (2)"));

  it("should preserve multiple internal spaces when appending suffix", () =>
    expect(deduplicate("Annual   Report", new Set(["Annual   Report"]))).toBe(
      "Annual   Report (1)",
    ));

  it("should handle unicode names when appending suffix", () =>
    expect(deduplicate("café", new Set(["café"]))).toBe("café (1)"));

  it("should not treat fullwidth parentheses/digits as numeric suffix", () =>
    expect(deduplicate("Report（1）", new Set(["Report（1）"]))).toBe(
      "Report（1） (1)",
    ));

  it("should handle emoji in names when appending suffix", () =>
    expect(deduplicate("Report 📄", new Set(["Report 📄"]))).toBe("Report 📄 (1)"));
});
