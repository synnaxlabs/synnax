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

describe("trimFileName", () => {
  it("strips a trailing .json extension", () => {
    expect(Import.trimFileName("dashboard.json")).toEqual("dashboard");
  });

  it("leaves names without a .json extension unchanged", () => {
    expect(Import.trimFileName("dashboard")).toEqual("dashboard");
    expect(Import.trimFileName("data.csv")).toEqual("data.csv");
  });

  it("only strips the final .json, not embedded ones", () => {
    expect(Import.trimFileName("a.json.json")).toEqual("a.json");
    expect(Import.trimFileName("a.json.csv")).toEqual("a.json.csv");
  });

  it("strips a bare .json down to an empty string", () => {
    expect(Import.trimFileName(".json")).toEqual("");
  });

  it("returns an empty string unchanged", () => {
    expect(Import.trimFileName("")).toEqual("");
  });
});
