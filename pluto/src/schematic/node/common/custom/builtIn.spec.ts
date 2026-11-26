// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Custom } from "@/schematic/node/common/custom";

const svg = "<svg/>";

describe("Custom.createBuiltIn", () => {
  it("should make a symbol with no family static", () => {
    expect(Custom.createBuiltIn({ svg, handles: [] })).toMatchObject({
      parts: [],
      states: [],
      commands: [],
      looks: [],
      interactions: [],
    });
  });

  it("should give a family with no commands a state but no click", () => {
    const spec = Custom.createBuiltIn({ svg, handles: [], family: "light" });
    expect(spec).toMatchObject({ commands: [], interactions: [] });
    expect(spec.states.map((s) => s.key)).toEqual(["active"]);
  });

  it("should give a family that clicks a next between its two commands", () => {
    const spec = Custom.createBuiltIn({ svg, handles: [], family: "valve" });
    expect(spec.interactions).toEqual([
      { type: "next", key: "click", commands: ["deactivate", "activate"] },
    ]);
  });

  it("should keep the looks and interactions that the params set", () => {
    const spec = Custom.createBuiltIn({
      svg,
      handles: [],
      family: "valve",
      looks: [],
      interactions: [],
    });
    expect(spec).toMatchObject({ looks: [], interactions: [] });
  });
});
