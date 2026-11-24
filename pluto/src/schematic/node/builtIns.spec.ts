// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { enumeration } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { color } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import type { FC, PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Node } from "@/schematic/node";
import { Custom } from "@/schematic/node/common/custom";
import { createAsyncSynnaxWrapper } from "@/testutil";

describe("Schematic.Node.BUILT_INS", () => {
  it("should hold a spec for each symbol that the Core lists", async () => {
    const builtins = await createTestClient().schematics.symbols.retrieveBuiltins();
    expect([...Node.BUILT_INS.keys()].sort()).toEqual(
      builtins.map((b) => b.name).sort(),
    );
  });

  it("should hold no spec for a name that only an object prototype has", () => {
    expect(Node.BUILT_INS.get("constructor")).toBeUndefined();
  });

  it("should give each symbol the states and commands the Core lists", async () => {
    const builtins = await createTestClient().schematics.symbols.retrieveBuiltins();
    for (const { name, states, commands } of builtins) {
      const spec = Node.BUILT_INS.get(name);
      expect({ name, states: spec?.states, commands: spec?.commands }).toEqual({
        name,
        states,
        commands,
      });
    }
  });

  it("should give each part selectors that select a shape of its drawing", () => {
    for (const [name, spec] of Node.BUILT_INS) {
      const svg = new DOMParser().parseFromString(spec.drawing.markup, "image/svg+xml");
      const selectors = spec.parts.flatMap((p) => p.selectors);
      expect({
        name,
        missing: selectors.filter((s) => svg.querySelector(s) == null),
      }).toEqual({ name, missing: [] });
    }
  });

  it("should give each drawing with detail lines the detail part", () => {
    for (const [name, spec] of Node.BUILT_INS)
      if (spec.drawing.markup.includes("pluto--detail"))
        expect({ name, parts: spec.parts.map((p) => p.key) }).toEqual({
          name,
          parts: expect.arrayContaining(["detail"]),
        });
  });

  it("should give each symbol a spec that the Core accepts", async () => {
    const client = createTestClient();
    const symbols = await client.schematics.symbols.create({
      symbols: [...Node.BUILT_INS].map(([name, data]) => ({ name, data })),
    });
    await client.schematics.symbols.delete(symbols.map((s) => s.key));
  });

  describe("looks", () => {
    const client = createTestClient();
    const RED = color.construct("#ff0000");
    const GREEN = color.construct("#00ff00");
    let wrapper: FC<PropsWithChildren>;
    let colored: enumeration.Key;

    // The built-in enumerations have no colors, so each state reads one that has.
    beforeAll(async () => {
      wrapper = await createAsyncSynnaxWrapper({ client });
      const created = await client.enumerations.create({
        name: "Built-in looks",
        entries: [
          { key: "off", value: 0, name: "Off", color: { type: "fixed", value: RED } },
          { key: "on", value: 1, name: "On", color: { type: "fixed", value: GREEN } },
        ],
      });
      colored = created.key;
    });

    it.each<[string, number, Custom.Paint[]]>([
      ["valve", 0, [{ stroke: RED }]],
      ["valve", 1, [{ stroke: GREEN }, { fill: GREEN }]],
      [
        "normally_open_solenoid_valve",
        0,
        [{ stroke: RED }, { selectors: [".pluto-body"], fill: RED }],
      ],
      [
        "normally_open_solenoid_valve",
        1,
        [
          { stroke: GREEN },
          { selectors: [".pluto-solenoid-valve__coil"], fill: GREEN },
        ],
      ],
      [
        "pump",
        1,
        [
          { stroke: GREEN },
          { fill: GREEN },
          { selectors: [".pluto--detail"], stroke: color.construct("#000000e6") },
        ],
      ],
      ["light", 1, [{ fill: GREEN }]],
      ["switch", 1, []],
      ["thruster", 1, []],
    ])(
      "should paint %s at %d as its old variant draws",
      async (name, value, paints) => {
        const spec = Node.BUILT_INS.get(name);
        if (spec == null) throw new Error(`expected a spec for ${name}`);
        const states = spec.states.map((s) =>
          s.type === "enumeration" ? { ...s, enumeration: colored } : s,
        );
        const { result } = renderHook(
          () => Custom.usePaints({ ...spec, states }, value),
          { wrapper },
        );
        await waitFor(() => expect(result.current).toEqual(paints));
      },
    );
  });
});
