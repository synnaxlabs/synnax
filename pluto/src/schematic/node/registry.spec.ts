// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, schematic } from "@synnaxlabs/client";
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

const createSymbol = (variant: string, stateKeys: string[]): schematic.symbol.Symbol =>
  schematic.symbol.symbolZ.parse({
    name: "Pump",
    data: {
      svg: "<svg></svg>",
      variant,
      states: stateKeys.map((key) => ({ key, name: key })),
    },
  });

describe("Schematic.Node.resolveSpec", () => {
  it("should return the registered spec for a variant", () => {
    expect(resolveSpec("tank")).toBe(REGISTRY.tank);
  });

  it("should throw NotFoundError for an unknown variant", () => {
    expect(() => resolveSpec("not-a-symbol")).toThrow(NotFoundError);
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
    expect(config.stalenessTimeout).toBe(5);
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
    expect(customVariant(createSymbol("static", ["base"]))).toBe("custom_static");
  });

  it("should draw an actuator with several states as a custom actuator", () => {
    const symbol = createSymbol("actuator", ["base", "active"]);
    expect(customVariant(symbol)).toBe("custom_actuator");
  });

  it("should draw an actuator with a single state as a custom static node", () => {
    expect(customVariant(createSymbol("actuator", ["base"]))).toBe("custom_static");
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
