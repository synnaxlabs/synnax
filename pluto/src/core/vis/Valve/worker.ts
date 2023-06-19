// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { AetherContext, AetherLeaf, Update } from "@/core/aether/worker";

export const valveState = z.object({
  triggered: z.boolean(),
  active: z.boolean(),
});

export class Valve extends AetherLeaf<typeof valveState> {
  static readonly TYPE = "valve";

  constructor(update: Update) {
    super(update, valveState);
  }

  handleUpdate(ctx: AetherContext): void {
    if (this.state.triggered) {
      setTimeout(() => {
        this.setState({ triggered: false, active: !this.state.active });
      }, 1000);
    }
  }

  render(): void {}
}
