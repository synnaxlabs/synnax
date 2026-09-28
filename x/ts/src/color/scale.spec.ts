// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { color } from "@/color";

describe("scale", () => {
  it("should default to no bands and stepped fills", () => {
    expect(color.scaleZ.parse({})).toEqual({ bands: [], smooth: false });
  });

  it("should parse a band's crude color and default it to steady", () => {
    const band = color.bandZ.parse({ key: "a", threshold: 10, color: "#ff0000" });
    expect(band).toEqual({
      key: "a",
      threshold: 10,
      color: color.construct("#ff0000"),
      flashing: false,
    });
  });
});
