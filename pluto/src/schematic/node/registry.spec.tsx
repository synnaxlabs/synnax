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
import { deep, TimeSpan } from "@synnaxlabs/x";
import { fireEvent } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { GroupBox } from "@/schematic/node/groupBox";
import {
  changeConfig,
  createConfig,
  CUSTOM_VARIANTS,
  customVariant,
  isConfig,
  isCustomConfig,
  isCustomVariant,
  REGISTRY,
  resolveSpec,
  STATIC_SPECS,
  symbolKey,
} from "@/schematic/node/registry";
import { renderForm } from "@/testutil";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const createSymbol = (variant: string): schematic.symbol.Symbol =>
  schematic.symbol.symbolZ.parse({
    name: "Pump",
    data: { drawing: { type: "svg", markup: "<svg></svg>" }, variant },
  });

describe("Schematic.Node.resolveSpec", () => {
  it("should return the registered spec for a variant", () => {
    expect(resolveSpec("tank")).toBe(REGISTRY.tank);
  });

  it("should throw NotFoundError for an unknown variant", () => {
    expect(() => resolveSpec("not-a-symbol")).toThrow(NotFoundError);
  });
});

describe("Schematic.Node.REGISTRY", () => {
  // The node cutover adds the drawing of normally_open_solenoid_valve.
  it("should draw every symbol that the Core ships", async () => {
    const builtins = await createTestClient().schematics.symbols.retrieveBuiltins();
    expect(builtins.map((b) => b.name).filter((name) => !(name in REGISTRY))).toEqual([
      "normally_open_solenoid_valve",
    ]);
  });
});

describe("Schematic.Node.isConfig", () => {
  it("should accept a node config", () => {
    expect(isConfig(createConfig({ variant: "tank" }))).toBe(true);
  });

  it("should accept a group box config", () => {
    expect(isConfig(createConfig({ variant: GroupBox.VARIANT }))).toBe(true);
  });

  it("should reject an edge config", () => {
    expect(isConfig(schematic.pipeEdgeConfigZ.parse({ variant: "pipe" }))).toBe(false);
  });
});

describe("Schematic.Node.isCustomVariant", () => {
  it("should accept every custom variant", () => {
    for (const variant of CUSTOM_VARIANTS) expect(isCustomVariant(variant)).toBe(true);
  });

  it("should reject a built-in variant", () => {
    expect(isCustomVariant("tank")).toBe(false);
  });

  it("should reject an unset variant", () => {
    expect(isCustomVariant(undefined)).toBe(false);
  });
});

describe("Schematic.Node.isCustomConfig", () => {
  it("should accept a custom symbol config", () => {
    const config = createConfig({ variant: "custom_static", specKey: "spec-1" });
    expect(isCustomConfig(config)).toBe(true);
  });

  it("should reject a built-in symbol config", () => {
    expect(isCustomConfig(createConfig({ variant: "tank" }))).toBe(false);
  });
});

describe("Schematic.Node.STATIC_SPECS", () => {
  it("should not include custom variants", () => {
    const keys = STATIC_SPECS.map((s) => s.key);
    for (const variant of CUSTOM_VARIANTS) expect(keys).not.toContain(variant);
  });

  it("should include all non-custom registry entries except groupBox", () => {
    const expectedKeys = Object.keys(REGISTRY).filter(
      (k) => !isCustomVariant(k) && k !== GroupBox.VARIANT,
    );
    const actualKeys = STATIC_SPECS.map((s) => s.key);
    expect(actualKeys).toEqual(expect.arrayContaining(expectedKeys));
    expect(actualKeys).toHaveLength(expectedKeys.length);
  });

  it("should not include groupBox", () => {
    const keys = STATIC_SPECS.map((s) => s.key);
    expect(keys).not.toContain(GroupBox.VARIANT);
  });

  it("groupBox should exist in the registry", () => {
    expect(REGISTRY).toHaveProperty(GroupBox.VARIANT);
  });

  it("custom variants should exist in the registry", () => {
    for (const variant of CUSTOM_VARIANTS) expect(REGISTRY).toHaveProperty(variant);
  });
});

describe("Schematic.Node.createConfig", () => {
  it("should fill the schema defaults for the variant", () => {
    const config = createConfig({ variant: "value" });
    expect(config.variant).toBe("value");
    expect(config.inlineSize).toBe(70);
    expect(config.stalenessTimeout).toBe(TimeSpan.seconds(5).nanoseconds);
  });

  it("should name the label after the spec when the input sets none", () => {
    const spec = resolveSpec("tank");
    expect(createConfig({ variant: "tank" }).label.label).toBe(spec.label ?? spec.name);
  });

  it("should keep a label the input sets", () => {
    const config = createConfig({ variant: "tank", label: { label: "T-100" } });
    expect(config.label.label).toBe("T-100");
  });

  it("should keep the input's fields over the defaults", () => {
    const config = createConfig({ variant: "value", channel: 42, units: "bar" });
    expect(config.channel).toBe(42);
    expect(config.units).toBe("bar");
  });

  it("should write the spec's value into a new text box", () => {
    expect(createConfig({ variant: "text_box" }).value).toBe("Text box");
  });

  it("should keep a text box value the input sets", () => {
    expect(createConfig({ variant: "text_box", value: "" }).value).toBe("");
  });

  it("should carry a custom symbol's spec key", () => {
    const config = createConfig({ variant: "custom_static", specKey: "spec-1" });
    expect(config.specKey).toBe("spec-1");
    expect(config.stateOverrides).toEqual([]);
  });

  it("should reject an unknown variant", () => {
    expect(() => createConfig({ variant: "not-a-symbol" } as never)).toThrow();
  });
});

describe("Schematic.Node.customVariant", () => {
  it("should draw a static symbol as a custom static node", () => {
    expect(customVariant(createSymbol("static"))).toBe("custom_static");
  });

  it("should draw an actuator as a custom actuator", () => {
    expect(customVariant(createSymbol("actuator"))).toBe("custom_actuator");
  });
});

describe("Schematic.Node.symbolKey", () => {
  it("should return the variant of a built-in symbol", () => {
    expect(symbolKey(createConfig({ variant: "tank" }))).toBe("tank");
  });

  it("should return the spec key of a custom symbol", () => {
    const config = createConfig({ variant: "custom_static", specKey: "spec-1" });
    expect(symbolKey(config)).toBe("spec-1");
  });
});

describe("Schematic.Node.changeConfig", () => {
  it("should keep the wrap width of a symbol that changes to a button", () => {
    const tank = createConfig({ variant: "tank", label: { maxInlineSize: 80 } });
    const changed = changeConfig(tank, { variant: "button" });
    expect(changed.label.maxInlineSize).toBe(80);
  });

  it("should keep the size of a field that changes", () => {
    const input = createConfig({ variant: "input", size: "large" });
    expect(changeConfig(input, { variant: "button" }).size).toBe("large");
  });

  it("should keep each field the new variant accepts", () => {
    const valve = createConfig({
      variant: "valve",
      label: { label: "OX-1" },
      orientation: "top",
      scale: 2,
      stateChannel: 12,
      commandChannel: 13,
    });
    const changed = changeConfig(valve, { variant: "solenoid_valve" });
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
    const valve = createConfig({ variant: "valve", stateChannel: 12 });
    const changed = changeConfig(valve, { variant: "tank" });
    expect(changed.variant).toBe("tank");
    expect(changed).not.toHaveProperty("stateChannel");
  });

  it("should fill the new variant's defaults", () => {
    const changed = changeConfig(createConfig({ variant: "valve" }), {
      variant: "value",
    });
    expect(changed.inlineSize).toBe(70);
  });

  it("should take the spec key of the new custom symbol", () => {
    const config = createConfig({
      variant: "custom_static",
      specKey: "spec-1",
      stateOverrides: [{ key: "base", name: "Base" }],
    });
    const changed = changeConfig(config, {
      variant: "custom_static",
      specKey: "spec-2",
    });
    expect(changed.specKey).toBe("spec-2");
    expect(changed.stateOverrides).toEqual([]);
  });

  it("should throw for an unknown variant", () => {
    const config = createConfig({ variant: "tank" });
    expect(() => changeConfig(config, { variant: "not-a-symbol" } as never)).toThrow();
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
      const { Form } = REGISTRY[variant];
      const { container, value } = renderForm(
        <Form tab={tab} />,
        createConfig({ variant }),
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
