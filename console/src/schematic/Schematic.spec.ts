// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Haul } from "@synnaxlabs/pluto";
import { describe, expect, it } from "vitest";

import {
  canDropSchematicHaulItem,
  createSymbolHaulItem,
  createValueHaulItem,
  filterSymbolHaulItems,
  filterValueHaulItems,
  isSchematicHaulItem,
  isSymbolHaulItem,
  isValueHaulItem,
  SYMBOL_HAUL_TYPE,
  VALUE_HAUL_TYPE,
  type ValueHaulData,
} from "@/schematic/Schematic";

const VARIANT = "valve";
const SPEC_KEY = "spec-1";
const VALUE_PROPS = {
  label: { label: "Pressure", level: "p" },
  color: "#ff0000",
} as ValueHaulData;

const OTHER: Haul.Item = { type: "other_type", key: "other" };

describe("schematic symbol haul utilities", () => {
  describe("createSymbolHaulItem", () => {
    it("creates an item with the symbol HAUL_TYPE", () => {
      expect(createSymbolHaulItem(VARIANT).type).toEqual(SYMBOL_HAUL_TYPE);
    });

    it("creates an item with the variant as the key", () => {
      expect(createSymbolHaulItem(VARIANT).key).toEqual(VARIANT);
    });

    it("creates an item with empty data when none is provided", () => {
      expect(createSymbolHaulItem(VARIANT).data).toEqual({});
    });

    it("creates an item with the provided specKey in data", () => {
      expect(createSymbolHaulItem(VARIANT, { specKey: SPEC_KEY }).data).toEqual({
        specKey: SPEC_KEY,
      });
    });
  });

  describe("isSymbolHaulItem", () => {
    it("returns true for a symbol item", () => {
      expect(isSymbolHaulItem(createSymbolHaulItem(VARIANT))).toBe(true);
    });

    it("returns false for a value item", () => {
      expect(isSymbolHaulItem(createValueHaulItem(VALUE_PROPS))).toBe(false);
    });

    it("returns false for an item of another kind", () => {
      expect(isSymbolHaulItem(OTHER)).toBe(false);
    });
  });

  describe("filterSymbolHaulItems", () => {
    it("keeps only symbol items", () => {
      const symbol = createSymbolHaulItem(VARIANT);
      const value = createValueHaulItem(VALUE_PROPS);
      expect(filterSymbolHaulItems([symbol, value, OTHER])).toEqual([symbol]);
    });
  });
});

describe("schematic value haul utilities", () => {
  describe("createValueHaulItem", () => {
    it("creates an item with the value HAUL_TYPE", () => {
      expect(createValueHaulItem(VALUE_PROPS).type).toEqual(VALUE_HAUL_TYPE);
    });

    it("creates an item with the literal 'value' as the key", () => {
      expect(createValueHaulItem(VALUE_PROPS).key).toEqual("value");
    });

    it("creates an item carrying the full value props in data", () => {
      expect(createValueHaulItem(VALUE_PROPS).data).toEqual(VALUE_PROPS);
    });
  });

  describe("isValueHaulItem", () => {
    it("returns true for a value item", () => {
      expect(isValueHaulItem(createValueHaulItem(VALUE_PROPS))).toBe(true);
    });

    it("returns false for a symbol item", () => {
      expect(isValueHaulItem(createSymbolHaulItem(VARIANT))).toBe(false);
    });

    it("returns false for an item of another kind", () => {
      expect(isValueHaulItem(OTHER)).toBe(false);
    });
  });

  describe("filterValueHaulItems", () => {
    it("keeps only value items", () => {
      const symbol = createSymbolHaulItem(VARIANT);
      const value = createValueHaulItem(VALUE_PROPS);
      expect(filterValueHaulItems([symbol, value, OTHER])).toEqual([value]);
    });
  });
});

describe("schematic combined haul utilities", () => {
  describe("isSchematicHaulItem", () => {
    it("returns true for a symbol item", () => {
      expect(isSchematicHaulItem(createSymbolHaulItem(VARIANT))).toBe(true);
    });

    it("returns true for a value item", () => {
      expect(isSchematicHaulItem(createValueHaulItem(VALUE_PROPS))).toBe(true);
    });

    it("returns false for an item of another kind", () => {
      expect(isSchematicHaulItem(OTHER)).toBe(false);
    });
  });

  describe("canDropSchematicHaulItem", () => {
    it("returns true when items contain a symbol item", () => {
      expect(
        canDropSchematicHaulItem({
          source: OTHER,
          items: [createSymbolHaulItem(VARIANT), OTHER],
        }),
      ).toBe(true);
    });

    it("returns true when items contain a value item", () => {
      expect(
        canDropSchematicHaulItem({
          source: OTHER,
          items: [createValueHaulItem(VALUE_PROPS), OTHER],
        }),
      ).toBe(true);
    });

    it("returns false when items contain neither a symbol nor a value item", () => {
      expect(canDropSchematicHaulItem({ source: OTHER, items: [OTHER] })).toBe(false);
    });
  });
});
