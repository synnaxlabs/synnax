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
  createConfig,
  CUSTOM_VARIANTS,
  isConfig,
  isCustomConfig,
  isCustomVariant,
  REGISTRY,
  resolveSpec,
  STATIC_SPECS,
} from "@/schematic/node/registry";

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
    expect(config.units).toBe("psi");
    expect(config.scale).toBe(1);
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
