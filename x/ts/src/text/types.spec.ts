// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { text } from "@/text";

describe("downLevel", () => {
  it("should step a level down by one rank", () => {
    expect(text.downLevel("h1")).toEqual("h2");
    expect(text.downLevel("h5")).toEqual("p");
  });

  it("should keep the smallest level", () => {
    expect(text.downLevel("tiny")).toEqual("tiny");
  });
});
