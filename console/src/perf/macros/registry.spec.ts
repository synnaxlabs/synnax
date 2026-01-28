// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  createMacroType,
  getAllMacroDefinitions,
  getAllMacroTypes,
  getMacro,
  getMacroDefinition,
  getMacrosByCategory,
  isMacroRegistered,
  type MacroCategory,
  registerMacro,
} from "@/perf/macros/registry";
import { type MacroStep } from "@/perf/macros/types";

const TEST_TYPE = createMacroType("testRegistryMacro");
const TEST_TYPE_2 = createMacroType("testRegistryMacro2");
const LINEPLOT_TYPE = createMacroType("testLinePlot");
const SCHEMATIC_TYPE = createMacroType("testSchematic");

const createMockSteps = (): MacroStep[] => [
  { name: "Step 1", execute: vi.fn().mockResolvedValue(undefined) },
];

describe("registry", () => {
  beforeEach(() => {
    registerMacro({
      type: TEST_TYPE,
      name: "Test Macro",
      description: "A test macro",
      category: "general" as MacroCategory,
      factory: createMockSteps,
    });
  });

  describe("registerMacro", () => {
    it("should register a macro definition", () => {
      expect(isMacroRegistered(TEST_TYPE)).toBe(true);
    });

    it("should overwrite existing macro with warning", () => {
      const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

      registerMacro({
        type: TEST_TYPE,
        name: "Overwritten Macro",
        description: "Overwritten",
        category: "general" as MacroCategory,
        factory: createMockSteps,
      });

      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining("already registered"),
      );
      expect(getMacroDefinition(TEST_TYPE)?.name).toBe("Overwritten Macro");

      consoleSpy.mockRestore();
    });
  });

  describe("getMacro", () => {
    it("should return steps from factory", () => {
      const steps = getMacro(TEST_TYPE);
      expect(steps).toHaveLength(1);
      expect(steps[0].name).toBe("Step 1");
    });

    it("should throw error for unknown macro type", () => {
      const unknownType = createMacroType("unknownMacro");
      expect(() => getMacro(unknownType)).toThrow("Unknown macro type");
    });
  });

  describe("getMacroDefinition", () => {
    it("should return definition for registered macro", () => {
      const def = getMacroDefinition(TEST_TYPE);
      expect(def).toBeDefined();
      expect(def?.name).toBe("Test Macro");
      expect(def?.description).toBe("A test macro");
    });

    it("should return undefined for unregistered macro", () => {
      const unknownType = createMacroType("unknownMacro");
      expect(getMacroDefinition(unknownType)).toBeUndefined();
    });
  });

  describe("getAllMacroTypes", () => {
    it("should return all registered macro types", () => {
      registerMacro({
        type: TEST_TYPE_2,
        name: "Test Macro 2",
        description: "Another test macro",
        category: "general" as MacroCategory,
        factory: createMockSteps,
      });

      const types = getAllMacroTypes();
      expect(types).toContain(TEST_TYPE);
      expect(types).toContain(TEST_TYPE_2);
    });
  });

  describe("getAllMacroDefinitions", () => {
    it("should return all registered definitions", () => {
      const definitions = getAllMacroDefinitions();
      expect(definitions.length).toBeGreaterThan(0);
      expect(definitions.some((d) => d.type === TEST_TYPE)).toBe(true);
    });
  });

  describe("getMacrosByCategory", () => {
    it("should filter macros by category", () => {
      registerMacro({
        type: LINEPLOT_TYPE,
        name: "LinePlot Test",
        description: "Test",
        category: "lineplot" as MacroCategory,
        factory: createMockSteps,
      });

      registerMacro({
        type: SCHEMATIC_TYPE,
        name: "Schematic Test",
        description: "Test",
        category: "schematic" as MacroCategory,
        factory: createMockSteps,
      });

      const linePlotMacros = getMacrosByCategory("lineplot");
      expect(linePlotMacros.some((m) => m.type === LINEPLOT_TYPE)).toBe(true);
      expect(linePlotMacros.some((m) => m.type === SCHEMATIC_TYPE)).toBe(false);
    });
  });

  describe("isMacroRegistered", () => {
    it("should return true for registered macros", () => {
      expect(isMacroRegistered(TEST_TYPE)).toBe(true);
    });

    it("should return false for unregistered macros", () => {
      const unknownType = createMacroType("unknownMacro");
      expect(isMacroRegistered(unknownType)).toBe(false);
    });
  });

  describe("createMacroType", () => {
    it("should create a branded MacroType from string", () => {
      const myType = createMacroType("myCustomType");
      expect(myType).toBe("myCustomType");
    });
  });
});
