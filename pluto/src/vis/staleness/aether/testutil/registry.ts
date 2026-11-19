// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z } from "zod";

import { aether } from "@/aether/aether";

/**
 * Returns a registry that mounts a stub in place of the component of the given type.
 * The stub parses its state with schema and reports stale on its first update.
 */
export const createRegistry = (
  type: string,
  schema: z.ZodType<{ stale: boolean }>,
): aether.ComponentRegistry => {
  class Stale extends aether.Leaf<typeof schema> {
    schema = schema;

    afterUpdate(): void {
      if (!this.state.stale) this.setState((s) => ({ ...s, stale: true }));
    }
  }
  return { [type]: Stale };
};
