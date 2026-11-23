// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { theme } from "@/theme";

const THEME = theme.themeZ.parse(theme.SYNNAX_LIGHT);

describe("fontString", () => {
  it("should draw numeric text in the tabular face at its weight", () => {
    expect(theme.fontString(THEME, { level: "p", weight: 450, numeric: true })).toMatch(
      /^ 450 [\d.]+px Inter Tabular, sans-serif$/,
    );
  });

  it("should draw code in the code face", () => {
    expect(
      theme.fontString(THEME, { level: "p", code: true, numeric: true }),
    ).toContain(THEME.typography.codeFamily);
  });
});
