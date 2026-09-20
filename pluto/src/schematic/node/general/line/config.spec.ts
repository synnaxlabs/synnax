// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { configZ, VARIANT } from "@/schematic/node/general/line/config";
import { defaultConfig } from "@/schematic/node/general/line/external";

describe("Line.configZ", () => {
  it("should accept the default config", () => {
    expect(configZ.safeParse(defaultConfig()).success).toBe(true);
  });

  it("should accept a config without a color or stroke width", () => {
    const config = { ...defaultConfig(), color: undefined, strokeWidth: undefined };
    expect(configZ.safeParse(config).success).toBe(true);
  });

  it("should require both endpoints", () => {
    expect(configZ.safeParse({ ...defaultConfig(), start: undefined }).success).toBe(
      false,
    );
    expect(configZ.safeParse({ ...defaultConfig(), end: undefined }).success).toBe(
      false,
    );
  });

  it("should reject an endpoint missing a coordinate", () => {
    const config = { ...defaultConfig(), end: { x: 100 } };
    expect(configZ.safeParse(config).success).toBe(false);
  });

  it("should reject another variant", () => {
    expect(configZ.safeParse({ ...defaultConfig(), variant: "box" }).success).toBe(
      false,
    );
  });

  it("should stamp the default config with the line variant", () => {
    expect(defaultConfig().variant).toBe(VARIANT);
  });
});
