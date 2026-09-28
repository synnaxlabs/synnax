// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { MAX_NAME_LENGTH, readName } from "@/server/license/machine";

describe("machine.readName", () => {
  it("should trim the name it is given", () => {
    expect(readName("  Test stand  ")).toBe("Test stand");
  });

  it("should cut a name longer than the bound", () => {
    expect(readName("a".repeat(MAX_NAME_LENGTH + 10))).toHaveLength(MAX_NAME_LENGTH);
  });

  it("should refuse a name that is blank or missing", () => {
    expect(() => readName("   ")).toThrow("The machine needs a name");
    expect(() => readName(undefined)).toThrow("The machine needs a name");
  });
});
