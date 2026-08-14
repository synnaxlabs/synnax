// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Import } from "@/platform/import";

describe("isParsableFile", () => {
  it("should accept plain .json files at any depth", () => {
    expect(Import.isParsableFile("Main.json")).toBe(true);
    expect(Import.isParsableFile("Propulsion/Tanks/Pressure.json")).toBe(true);
  });
  it("should reject files without a .json extension", () => {
    expect(Import.isParsableFile(".DS_Store")).toBe(false);
    expect(Import.isParsableFile("Propulsion/.DS_Store")).toBe(false);
    expect(Import.isParsableFile("notes.txt")).toBe(false);
  });
  it("should reject dot files even with a .json extension", () => {
    expect(Import.isParsableFile("__MACOSX/._Main.json")).toBe(false);
    expect(Import.isParsableFile("._Main.json")).toBe(false);
  });
});
