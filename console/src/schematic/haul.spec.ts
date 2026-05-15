// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x/id";
import type { Haul } from "@synnaxlabs/charon/haul";

import { describe, expect, it } from "vitest";

import { Schematic } from "@/schematic";
import { type AddNodeProps } from "@/schematic/symbols/useAddNode";

const VARIANT = "valve";
const BASE_ADD_PROPS: AddNodeProps = {
  key: id.create(),
  variant: VARIANT,
};
const SPEC_KEY = "spec-1";
const VALUE_PROPS: Schematic.ValueHaulData = {
  variant: "value",
  label: { label: "Pressure", level: "p" },
  color: "#ff0000",
};

const OTHER: Haul.Item = { type: "other_type", key: "other" };

describe("schematic symbol haul utilities", () => {
  describe("createSymbolHaulItem", () => {
    it("creates an item with the symbol HAUL_TYPE", () => {
      expect(Schematic.createSymbolHaulItem(BASE_ADD_PROPS).type).toEqual(
        Schematic.SYMBOL_HAUL_TYPE,
      );
    });

    it("creates an item with the variant as the key", () => {
      expect(Schematic.createSymbolHaulItem(BASE_ADD_PROPS).key).toEqual(
        BASE_ADD_PROPS.key,
      );
    });

    it("creates an item with empty data when none is provided", () => {
      expect(Schematic.createSymbolHaulItem(BASE_ADD_PROPS).data).toEqual(
        BASE_ADD_PROPS,
      );
    });

    it("creates an item with the provided specKey in data", () => {
      expect(
        Schematic.createSymbolHaulItem({ ...BASE_ADD_PROPS, specKey: SPEC_KEY }).data,
      ).toEqual({
        key: BASE_ADD_PROPS.key,
        specKey: SPEC_KEY,
        variant: "valve",
      });
    });
  });

  describe("isSymbolHaulItem", () => {
    it("returns true for a symbol item", () => {
      expect(
        Schematic.isSymbolHaulItem(Schematic.createSymbolHaulItem(BASE_ADD_PROPS)),
      ).toBe(true);
    });

    it("returns false for a value item", () => {
      expect(
        Schematic.isSymbolHaulItem(Schematic.createValueHaulItem(VALUE_PROPS)),
      ).toBe(false);
    });

    it("returns false for an item of another kind", () => {
      expect(Schematic.isSymbolHaulItem(OTHER)).toBe(false);
    });
  });

  describe("filterSymbolHaulItems", () => {
    it("keeps only symbol items", () => {
      const symbol = Schematic.createSymbolHaulItem(BASE_ADD_PROPS);
      const value = Schematic.createValueHaulItem(VALUE_PROPS);
      expect(Schematic.filterSymbolHaulItems([symbol, value, OTHER])).toEqual([symbol]);
    });
  });
});

describe("schematic value haul utilities", () => {
  describe("createValueHaulItem", () => {
    it("creates an item with the value HAUL_TYPE", () => {
      expect(Schematic.createValueHaulItem(VALUE_PROPS).type).toEqual(
        Schematic.VALUE_HAUL_TYPE,
      );
    });

    it("creates an item with the literal 'value' as the key", () => {
      expect(Schematic.createValueHaulItem(VALUE_PROPS).key).toEqual("value");
    });

    it("creates an item carrying the full value props in data", () => {
      expect(Schematic.createValueHaulItem(VALUE_PROPS).data).toEqual(VALUE_PROPS);
    });
  });

  describe("isValueHaulItem", () => {
    it("returns true for a value item", () => {
      expect(
        Schematic.isValueHaulItem(Schematic.createValueHaulItem(VALUE_PROPS)),
      ).toBe(true);
    });

    it("returns false for a symbol item", () => {
      expect(
        Schematic.isValueHaulItem(Schematic.createSymbolHaulItem(BASE_ADD_PROPS)),
      ).toBe(false);
    });

    it("returns false for an item of another kind", () => {
      expect(Schematic.isValueHaulItem(OTHER)).toBe(false);
    });
  });

  describe("filterValueHaulItems", () => {
    it("keeps only value items", () => {
      const symbol = Schematic.createSymbolHaulItem(BASE_ADD_PROPS);
      const value = Schematic.createValueHaulItem(VALUE_PROPS);
      expect(Schematic.filterValueHaulItems([symbol, value, OTHER])).toEqual([value]);
    });
  });
});

describe("schematic combined haul utilities", () => {
  describe("isSchematicHaulItem", () => {
    it("returns true for a symbol item", () => {
      expect(Schematic.isHaulItem(Schematic.createSymbolHaulItem(BASE_ADD_PROPS))).toBe(
        true,
      );
    });

    it("returns true for a value item", () => {
      expect(Schematic.isHaulItem(Schematic.createValueHaulItem(VALUE_PROPS))).toBe(
        true,
      );
    });

    it("returns false for an item of another kind", () => {
      expect(Schematic.isHaulItem(OTHER)).toBe(false);
    });
  });

  describe("canDropSchematicHaulItem", () => {
    it("returns true when items contain a symbol item", () => {
      expect(
        Schematic.canDropHaulItem({
          source: OTHER,
          items: [Schematic.createSymbolHaulItem(BASE_ADD_PROPS), OTHER],
        }),
      ).toBe(true);
    });

    it("returns true when items contain a value item", () => {
      expect(
        Schematic.canDropHaulItem({
          source: OTHER,
          items: [Schematic.createValueHaulItem(VALUE_PROPS), OTHER],
        }),
      ).toBe(true);
    });

    it("returns false when items contain neither a symbol nor a value item", () => {
      expect(Schematic.canDropHaulItem({ source: OTHER, items: [OTHER] })).toBe(false);
    });
  });
});
