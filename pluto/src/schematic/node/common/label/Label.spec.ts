// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Label } from "@/schematic/node/common/label";

describe("Label.defaultConfig", () => {
  it("should populate the supplied label and apply standard defaults", () => {
    expect(Label.defaultConfig("My Symbol")).toEqual({
      label: "My Symbol",
      level: "h5",
      orientation: "top",
      maxInlineSize: 150,
      align: "center",
      direction: "x",
    });
  });

  it("should preserve an empty label string when explicitly provided", () => {
    expect(Label.defaultConfig("").label).toBe("");
  });

  it("should produce an object that round-trips through configZ", () => {
    expect(() => Label.configZ.parse(Label.defaultConfig("anything"))).not.toThrow();
  });
});

describe("Label.configZ", () => {
  it("should accept an empty object since every field is optional", () => {
    expect(() => Label.configZ.parse({})).not.toThrow();
  });

  it("should accept a fully-populated config", () => {
    expect(() =>
      Label.configZ.parse({
        label: "Tag-101",
        level: "p",
        orientation: "bottom",
        direction: "y",
        maxInlineSize: 250,
        align: "start",
      }),
    ).not.toThrow();
  });

  it("should reject non-string label values", () => {
    expect(() => Label.configZ.parse({ label: 123 })).toThrow();
  });

  it("should reject unknown locations on orientation", () => {
    expect(() => Label.configZ.parse({ orientation: "diagonal" })).toThrow();
  });

  it("should reject non-numeric maxInlineSize", () => {
    expect(() => Label.configZ.parse({ maxInlineSize: "150px" })).toThrow();
  });
});

describe("Label.labeledConfigZ", () => {
  it("should accept a config with both an embedded label and an outer orientation", () => {
    expect(() =>
      Label.labeledConfigZ.parse({
        label: { label: "valve" },
        orientation: "top",
      }),
    ).not.toThrow();
  });

  it("should accept an empty config", () => {
    expect(() => Label.labeledConfigZ.parse({})).not.toThrow();
  });

  it("should reject inner labels that violate configZ", () => {
    expect(() =>
      Label.labeledConfigZ.parse({ label: { level: "not-a-level" } }),
    ).toThrow();
  });

  it("should reject orientations that are not Outer locations", () => {
    expect(() => Label.labeledConfigZ.parse({ orientation: "center" })).toThrow();
  });
});
