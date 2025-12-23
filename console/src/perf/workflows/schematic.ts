// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { registerWorkflow } from "@/perf/workflows/registry";
import { type WorkflowContext, type WorkflowStep } from "@/perf/workflows/types";
import { Schematic } from "@/schematic";

/**
 * Workflow to create a new schematic.
 */
export const createSchematicWorkflow = (): WorkflowStep[] => [
  {
    name: "Create Schematic",
    execute: async (ctx: WorkflowContext) => {
      const timestamp = Date.now();
      const { key } = ctx.placer(
        Schematic.create({
          name: `Perf Test Schematic ${timestamp}`,
          location: "mosaic",
        }),
      );
      ctx.createdLayoutKeys.push(key);
    },
    delayAfterMs: 500,
  },
];

registerWorkflow({
  type: "createSchematic",
  name: "Create Schematic",
  description: "Creates a new schematic diagram",
  category: "schematic",
  factory: createSchematicWorkflow,
});
