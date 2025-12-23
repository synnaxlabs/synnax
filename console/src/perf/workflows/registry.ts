// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type WorkflowStep, type WorkflowType } from "@/perf/workflows/types";

export type WorkflowCategory = "lineplot" | "schematic" | "table" | "general";

export interface WorkflowDefinition {
  type: WorkflowType;
  name: string;
  description: string;
  category: WorkflowCategory;
  factory: () => WorkflowStep[];
}

type WorkflowRegistry = Map<WorkflowType, WorkflowDefinition>;

const registry: WorkflowRegistry = new Map();

export const registerWorkflow = (definition: WorkflowDefinition): void => {
  if (registry.has(definition.type))
    console.warn(`Workflow "${definition.type}" is already registered, overwriting`);
  registry.set(definition.type, definition);
};

export const getWorkflow = (type: WorkflowType): WorkflowStep[] => {
  const definition = registry.get(type);
  if (definition == null) throw new Error(`Unknown workflow type: ${type}`);
  return definition.factory();
};

export const getWorkflowDefinition = (
  type: WorkflowType,
): WorkflowDefinition | undefined => registry.get(type);

export const getAllWorkflowTypes = (): WorkflowType[] =>
  Array.from(registry.keys());

export const getAllWorkflowDefinitions = (): WorkflowDefinition[] =>
  Array.from(registry.values());

export const getWorkflowsByCategory = (
  category: WorkflowCategory,
): WorkflowDefinition[] =>
  Array.from(registry.values()).filter((def) => def.category === category);

export const isWorkflowRegistered = (type: WorkflowType): boolean =>
  registry.has(type);

/**
 * Creates a WorkflowType from a string. Use this when defining new workflow types
 * outside of the built-in types.
 *
 * @example
 * const MY_WORKFLOW_TYPE = createWorkflowType("myCustomWorkflow");
 *
 * registerWorkflow({
 *   type: MY_WORKFLOW_TYPE,
 *   name: "My Custom Workflow",
 *   ...
 * });
 */
export const createWorkflowType = (type: string): WorkflowType => type;
