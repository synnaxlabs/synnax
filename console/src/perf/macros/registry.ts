// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type MacroStep, type MacroType } from "@/perf/macros/types";

export type MacroCategory = "lineplot" | "schematic" | "table" | "general";

export interface MacroDefinition {
  type: MacroType;
  name: string;
  description: string;
  category: MacroCategory;
  factory: () => MacroStep[];
}

type MacroRegistry = Map<MacroType, MacroDefinition>;

const registry: MacroRegistry = new Map();

export const registerMacro = (definition: MacroDefinition): void => {
  if (registry.has(definition.type))
    console.warn(`Macro "${definition.type}" is already registered, overwriting`);
  registry.set(definition.type, definition);
};

export const getMacro = (type: MacroType): MacroStep[] => {
  const definition = registry.get(type);
  if (definition == null) throw new Error(`Unknown macro type: ${type}`);
  return definition.factory();
};

export const getMacroDefinition = (type: MacroType): MacroDefinition | undefined =>
  registry.get(type);

export const getAllMacroTypes = (): MacroType[] => Array.from(registry.keys());

export const getAllMacroDefinitions = (): MacroDefinition[] =>
  Array.from(registry.values());

export const getMacrosByCategory = (category: MacroCategory): MacroDefinition[] =>
  Array.from(registry.values()).filter((def) => def.category === category);

export const isMacroRegistered = (type: MacroType): boolean => registry.has(type);

/**
 * Creates a MacroType from a string. Use this when defining new macro types
 * outside of the built-in types.
 *
 * @example
 * const MY_MACRO_TYPE = createMacroType("myCustomMacro");
 *
 * registerMacro({
 *   type: MY_MACRO_TYPE,
 *   name: "My Custom Macro",
 *   ...
 * });
 */
export const createMacroType = <T extends string>(type: T): MacroType & T =>
  type as MacroType & T;
