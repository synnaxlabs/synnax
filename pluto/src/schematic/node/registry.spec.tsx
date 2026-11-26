// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { deep, TimeSpan, uuid } from "@synnaxlabs/x";
import { fireEvent } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Node } from "@/schematic/node";
import { GroupBox } from "@/schematic/node/groupBox";
import { createSynnaxWrapper, renderForm } from "@/testutil";

describe("Schematic.Node.resolveSpec", () => {
  it("should return the registered spec for a variant", () => {
    expect(Node.resolveSpec("tank")).toBe(Node.REGISTRY.tank);
  });

  it("should throw NotFoundError for an unknown variant", () => {
    expect(() => Node.resolveSpec("not-a-symbol")).toThrow(NotFoundError);
  });
});

describe("Schematic.Node.REGISTRY", () => {
  // The node cutover adds the drawing of normally_open_solenoid_valve.
  it("should draw every symbol that the Core ships", async () => {
    const builtins = await createTestClient().schematics.symbols.retrieveBuiltins();
    expect(
      builtins.map((b) => b.name).filter((name) => !(name in Node.REGISTRY)),
    ).toEqual(["normally_open_solenoid_valve"]);
  });
});

describe("Schematic.Node.isConfig", () => {
  it("should accept a node config", () => {
    expect(Node.isConfig(Node.createConfig({ variant: "tank" }))).toBe(true);
  });

  it("should accept a group box config", () => {
    expect(Node.isConfig(Node.createConfig({ variant: GroupBox.VARIANT }))).toBe(true);
  });

  it("should reject an edge config", () => {
    expect(Node.isConfig(schematic.pipeEdgeConfigZ.parse({ variant: "pipe" }))).toBe(
      false,
    );
  });
});

describe("Schematic.Node.STATIC_SPECS", () => {
  it("should include all registry entries except groupBox and symbol", () => {
    const expectedKeys = Object.keys(Node.REGISTRY).filter(
      (k) => k !== GroupBox.VARIANT && k !== "symbol",
    );
    const actualKeys = Node.STATIC_SPECS.map((s) => s.key);
    expect(actualKeys).toEqual(expect.arrayContaining(expectedKeys));
    expect(actualKeys).toHaveLength(expectedKeys.length);
  });

  it("should not include groupBox", () => {
    const keys = Node.STATIC_SPECS.map((s) => s.key);
    expect(keys).not.toContain(GroupBox.VARIANT);
  });

  it("should not include the symbol variant", () => {
    expect(Node.STATIC_SPECS.map((s) => s.key)).not.toContain("symbol");
  });

  it("groupBox should exist in the registry", () => {
    expect(Node.REGISTRY).toHaveProperty(GroupBox.VARIANT);
  });
});

describe("Schematic.Node.createConfig", () => {
  it("should fill the schema defaults for the variant", () => {
    const config = Node.createConfig({ variant: "value" });
    expect(config.variant).toBe("value");
    expect(config.inlineSize).toBe(70);
    expect(config.stalenessTimeout).toBe(TimeSpan.seconds(5).nanoseconds);
  });

  it("should name the label after the spec when the input sets none", () => {
    const spec = Node.resolveSpec("tank");
    expect(Node.createConfig({ variant: "tank" }).label.label).toBe(
      spec.label ?? spec.name,
    );
  });

  it("should keep a label the input sets", () => {
    const config = Node.createConfig({ variant: "tank", label: { label: "T-100" } });
    expect(config.label.label).toBe("T-100");
  });

  it("should keep the input's fields over the defaults", () => {
    const config = Node.createConfig({ variant: "value", channel: 42, units: "bar" });
    expect(config.channel).toBe(42);
    expect(config.units).toBe("bar");
  });

  it("should write the spec's value into a new text box", () => {
    expect(Node.createConfig({ variant: "text_box" }).value).toBe("Text box");
  });

  it("should keep a text box value the input sets", () => {
    expect(Node.createConfig({ variant: "text_box", value: "" }).value).toBe("");
  });

  it("should reject an unknown variant", () => {
    expect(() => Node.createConfig({ variant: "not-a-symbol" } as never)).toThrow();
  });
});

describe("Schematic.Node.symbolKey", () => {
  it("should return the variant of a built-in symbol", () => {
    expect(Node.symbolKey(Node.createConfig({ variant: "tank" }))).toBe("tank");
  });

  it("should return the name of the built-in that a symbol node draws", () => {
    const config = Node.createConfig({
      variant: "symbol",
      spec: { type: "builtin", name: "ball_valve" },
    });
    expect(Node.symbolKey(config)).toBe("ball_valve");
  });

  it("should return the key of the custom symbol that a symbol node draws", () => {
    const key = uuid.create();
    const config = Node.createConfig({
      variant: "symbol",
      spec: { type: "custom", key },
    });
    expect(Node.symbolKey(config)).toBe(key);
  });

  it("should return the key of a missing symbol that a symbol node draws", () => {
    const config = Node.createConfig({
      variant: "symbol",
      spec: { type: "missing", key: "gone" },
    });
    expect(Node.symbolKey(config)).toBe("gone");
  });
});

describe("Schematic.Node.builtInLabel", () => {
  it("should return the label of the variant that draws the built-in", () => {
    expect(Node.builtInLabel("valve")).toBe("Valve");
  });

  it("should label a built-in that no variant draws", () => {
    expect(Node.builtInLabel("normally_open_solenoid_valve")).toBe(
      "Normally open solenoid",
    );
  });

  it("should label every built-in", () => {
    for (const name of Node.BUILT_INS.keys())
      expect(Node.builtInLabel(name)).toBeDefined();
  });

  it.each(["not_a_symbol", "constructor", "symbol"])(
    "should return undefined for %s, which no built-in has",
    (name) => expect(Node.builtInLabel(name)).toBeUndefined(),
  );
});

describe("Schematic.Node.builtInName", () => {
  it("should return the name of the variant that draws the built-in", () => {
    expect(Node.builtInName("angled_valve")).toBe("Angled");
  });

  it("should name a built-in that no variant draws", () => {
    expect(Node.builtInName("normally_open_solenoid_valve")).toBe(
      "Normally open solenoid",
    );
  });

  it("should name every built-in", () => {
    for (const name of Node.BUILT_INS.keys())
      expect(Node.builtInName(name)).toBeDefined();
  });

  it.each(["not_a_symbol", "constructor", "symbol"])(
    "should return undefined for %s, which no built-in has",
    (name) => expect(Node.builtInName(name)).toBeUndefined(),
  );
});

describe("Schematic.Node.changeConfig", () => {
  it("should keep the wrap width of a symbol that changes to a button", () => {
    const tank = Node.createConfig({ variant: "tank", label: { maxInlineSize: 80 } });
    const changed = Node.changeConfig(tank, { variant: "button" });
    expect(changed.label.maxInlineSize).toBe(80);
  });

  it("should keep the size of a field that changes", () => {
    const input = Node.createConfig({ variant: "input", size: "large" });
    expect(Node.changeConfig(input, { variant: "button" }).size).toBe("large");
  });

  it("should keep each field the new variant accepts", () => {
    const valve = Node.createConfig({
      variant: "valve",
      label: { label: "OX-1" },
      orientation: "top",
      scale: 2,
      stateChannel: 12,
      commandChannel: 13,
    });
    const changed = Node.changeConfig(valve, { variant: "solenoid_valve" });
    expect(changed).toMatchObject({
      variant: "solenoid_valve",
      label: { label: "OX-1" },
      orientation: "top",
      scale: 2,
      stateChannel: 12,
      commandChannel: 13,
      normallyOpen: false,
    });
  });

  it("should drop the fields the new variant does not accept", () => {
    const valve = Node.createConfig({ variant: "valve", stateChannel: 12 });
    const changed = Node.changeConfig(valve, { variant: "tank" });
    expect(changed.variant).toBe("tank");
    expect(changed).not.toHaveProperty("stateChannel");
  });

  it("should fill the new variant's defaults", () => {
    const changed = Node.changeConfig(Node.createConfig({ variant: "valve" }), {
      variant: "value",
    });
    expect(changed.inlineSize).toBe(70);
  });

  it("should take the spec of the new symbol node and keep its bindings", () => {
    const config = Node.createConfig({
      variant: "symbol",
      spec: { type: "builtin", name: "valve" },
      states: [{ key: "active", source: { type: "channel", key: 12 } }],
      commands: [{ key: "open", channel: 13 }],
    });
    const changed = Node.changeConfig(config, {
      variant: "symbol",
      spec: { type: "builtin", name: "ball_valve" },
    });
    expect(changed).toMatchObject({
      spec: { type: "builtin", name: "ball_valve" },
      states: [{ key: "active", source: { type: "channel", key: 12 } }],
      commands: [{ key: "open", channel: 13 }],
    });
  });

  it("should throw for an unknown variant", () => {
    const config = Node.createConfig({ variant: "tank" });
    expect(() =>
      Node.changeConfig(config, { variant: "not-a-symbol" } as never),
    ).toThrow();
  });
});

const fieldPath = deep.path<Record<string, number>>();

describe("Schematic.Node.REGISTRY forms", () => {
  it.each([
    {
      variant: "circle",
      tab: undefined,
      label: "Radius",
      field: "radius",
      typed: "80.6",
      expected: 81,
    },
    {
      variant: "gauge",
      tab: "style",
      label: "Bar width",
      field: "barWidth",
      typed: "9.6",
      expected: 10,
    },
    {
      variant: "text_box",
      tab: undefined,
      label: "Wrap width",
      field: "inlineSize",
      typed: "80.6",
      expected: 81,
    },
    {
      variant: "box",
      tab: undefined,
      label: "Border radius",
      field: "borderRadius",
      typed: "80.6",
      expected: 81,
    },
    {
      variant: "value",
      tab: "style",
      label: "Width",
      field: "inlineSize",
      typed: "80.6",
      expected: 81,
    },
    {
      variant: "value",
      tab: "telemetry",
      label: "Averaging window",
      field: "rollingAverage",
      typed: "80.6",
      expected: 81,
    },
    {
      variant: "scale",
      tab: "telemetry",
      label: "Averaging window",
      field: "rollingAverage",
      typed: "80.6",
      expected: 81,
    },
  ] as const)(
    "should store a whole $label for a $variant",
    ({ variant, tab, label, field, typed, expected }) => {
      const { Form } = Node.REGISTRY[variant];
      const { container, value } = renderForm(
        <Form tab={tab} />,
        Node.createConfig({ variant }),
        schematic.nodeConfigZ,
        createSynnaxWrapper({ client: null }),
      );
      const input = Array.from(container.querySelectorAll(".lyra-input__item"))
        .find((el) => el.querySelector("label")?.textContent?.trim() === label)
        ?.querySelector("input");
      if (input == null) throw new Error(`expected a ${label} input`);
      fireEvent.change(input, { target: { value: typed } });
      fireEvent.blur(input);
      expect(value(fieldPath[field])).toBe(expected);
    },
  );
});
