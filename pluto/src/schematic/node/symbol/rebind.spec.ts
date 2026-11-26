// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { Node } from "@/schematic/node";

type SpecInput = schematic.symbol.New["data"];

const createSpec = (
  states: SpecInput["states"],
  commands: SpecInput["commands"] = [],
): schematic.symbol.Spec =>
  schematic.symbol.specZ.parse({
    drawing: { type: "svg", markup: "<svg/>" },
    variant: "static",
    states,
    commands,
  });

const FIXED = { type: "fixed" as const, value: 1 };
const ENUMERATION = createSpec([{ type: "enumeration", key: "mode", name: "Mode" }]);
const NUMBER = createSpec([{ type: "number", key: "mode", name: "Mode" }]);
const SOURCE: schematic.Source = { type: "channel", key: 5, command: 6 };
const BOUND: schematic.StateBinding = {
  key: "mode",
  source: SOURCE,
  default: 1,
  units: "psi",
};

describe("Schematic.Node.rebind", () => {
  it("should keep a binding whole when its state keeps its type", () => {
    const to = createSpec([
      { type: "enumeration", key: "mode", name: "Mode" },
      { type: "number", key: "level", name: "Level" },
    ]);
    expect(Node.rebind({ states: [BOUND], commands: [] }, ENUMERATION, to)).toEqual({
      states: [BOUND],
      commands: [],
    });
  });

  it("should keep only the channel source when the state changes type", () => {
    expect(
      Node.rebind({ states: [BOUND], commands: [] }, ENUMERATION, NUMBER).states,
    ).toEqual([{ key: "mode", source: SOURCE }]);
  });

  it("should drop a local source when the state changes type", () => {
    const local: schematic.StateBinding = {
      key: "mode",
      source: { type: "local", value: 1 },
    };
    expect(
      Node.rebind({ states: [local], commands: [] }, ENUMERATION, NUMBER).states,
    ).toEqual([]);
  });

  it("should drop a binding whose state the new spec lacks", () => {
    const to = createSpec([{ type: "enumeration", key: "other", name: "Other" }]);
    expect(Node.rebind({ states: [BOUND], commands: [] }, ENUMERATION, to)).toEqual({
      states: [],
      commands: [],
    });
  });

  it("should keep only the channel source when the old spec is missing", () => {
    expect(
      Node.rebind({ states: [BOUND], commands: [] }, undefined, ENUMERATION).states,
    ).toEqual([{ key: "mode", source: SOURCE }]);
  });

  it("should keep a command binding only when the new spec has its command", () => {
    const from = createSpec(
      [],
      [
        { key: "open", name: "Open", value: FIXED },
        { key: "vent", name: "Vent", value: FIXED },
      ],
    );
    const to = createSpec([], [{ key: "open", name: "Open", value: FIXED }]);
    const commands = [
      { key: "open", channel: 7 },
      { key: "vent", channel: 8 },
    ];
    expect(Node.rebind({ states: [], commands }, from, to).commands).toEqual([
      { key: "open", channel: 7 },
    ]);
  });

  it("should drop a command binding when the new command writes a state", () => {
    const to = createSpec(
      [{ type: "number", key: "mode", name: "Mode" }],
      [{ key: "open", name: "Open", state: "mode", value: FIXED }],
    );
    const commands = [{ key: "open", channel: 7 }];
    expect(Node.rebind({ states: [], commands }, NUMBER, to).commands).toEqual([]);
  });
});
