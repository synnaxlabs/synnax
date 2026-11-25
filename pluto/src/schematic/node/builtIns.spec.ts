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

  it("should let a look fill each shape that reads the symbol color", () => {
    // A look cannot fill a text over the theme, so it fills a span in the text.
    const paintable = (el: Element): boolean =>
      el.tagName === "text"
        ? el.querySelector(":scope > tspan.pluto--solid") != null
        : el.matches(".pluto--solid[fill='currentColor']");
    for (const [name, spec] of Node.BUILT_INS) {
      const svg = new DOMParser().parseFromString(spec.drawing.markup, "image/svg+xml");
      const reads = [...svg.querySelectorAll("[style*='--pluto-symbol']")];
      expect({
        name,
        unpaintable: reads.filter((el) => !paintable(el)),
        // A browser ignores a variable in a fill attribute.
        variable: svg.querySelector("[fill*='var(']"),
      }).toEqual({ name, unpaintable: [], variable: null });
      if (reads.length > 0)
        expect({ name, parts: spec.parts.map((p) => p.key) }).toEqual({
          name,
          parts: expect.arrayContaining(["solid"]),
        });
    }
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
    // The light theme shows GREEN darker, so it reads on the background.
    const BANDED_GREEN = color.construct("#00e800");
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

    const SOLID = ".pluto--solid";

    const expectPaints = async (
      name: string,
      value: number,
      paints: Custom.Paint[],
      nodeColor?: color.Color,
    ) => {
      const spec = Node.BUILT_INS.get(name);
      if (spec == null) throw new Error(`expected a spec for ${name}`);
      const states = spec.states.map((s) =>
        s.type === "enumeration" ? { ...s, enumeration: colored } : s,
      );
      const { result } = renderHook(
        () =>
          Custom.usePaints({ ...spec, states }, value, { builtIn: true, nodeColor }),
        { wrapper },
      );
      await waitFor(() => expect(result.current).toEqual(paints));
    };

    it.each<[string, number, Custom.Paint[]]>([
      ["valve", 0, [{ stroke: RED }]],
      ["valve", 1, [{ stroke: BANDED_GREEN }, { fill: BANDED_GREEN }]],
      [
        "normally_open_solenoid_valve",
        0,
        [{ stroke: RED }, { selectors: [".pluto-body"], fill: RED }],
      ],
      [
        "normally_open_solenoid_valve",
        1,
        [
          { stroke: BANDED_GREEN },
          { selectors: [".pluto-solenoid-valve__coil"], fill: BANDED_GREEN },
        ],
      ],
      [
        "pump",
        1,
        [
          { stroke: BANDED_GREEN },
          { fill: BANDED_GREEN },
          { selectors: [".pluto--detail"], stroke: color.construct("#000000e6") },
        ],
      ],
      [
        "butterfly_valve_one",
        1,
        [
          { stroke: BANDED_GREEN },
          { fill: BANDED_GREEN },
          { selectors: [SOLID], fill: BANDED_GREEN },
        ],
      ],
      ["light", 1, [{ fill: BANDED_GREEN }]],
      ["switch", 1, []],
      ["thruster", 1, []],
    ])(
      "should paint %s at %d as its old variant draws",
      async (name, value, paints) => {
        await expectPaints(name, value, paints);
      },
    );

    const STATIC_SOLID = [
      "flowmeter_coriolis",
      "flowmeter_electromagnetic",
      "flowmeter_float_sensor",
      "flowmeter_general",
      "flowmeter_nozzle",
      "flowmeter_orifice",
      "flowmeter_positive_displacement",
      "flowmeter_pulse",
      "flowmeter_ring_piston",
      "flowmeter_turbine",
      "flowmeter_variable_area",
      "flowmeter_venturi",
      "burst_disc",
      "check_valve_with_arrow",
      "iso_check_valve",
      "cross_junction",
      "t_junction",
    ];
    const ACTIVE_SOLID = [
      "angled_spring_loaded_relief_valve",
      "breather_valve",
      "butterfly_valve_one",
      "butterfly_valve_two",
      "four_way_valve",
      "needle_valve",
      "spring_loaded_relief_valve",
    ];

    it.each(STATIC_SOLID)(
      "should fill the solid shapes of %s with the node color",
      async (name) => {
        await expectPaints(name, 0, [{ selectors: [SOLID], fill: RED }], RED);
      },
    );

    it.each(STATIC_SOLID)(
      "should leave the solid shapes of %s to the theme with no node color",
      async (name) => {
        await expectPaints(name, 0, [{ selectors: [SOLID] }]);
      },
    );

    it.each(ACTIVE_SOLID)(
      "should fill the solid shapes of %s with the color of its stroke",
      async (name) => {
        await expectPaints(name, 0, [
          { stroke: RED },
          { selectors: [SOLID], fill: RED },
        ]);
      },
    );
  });
});
