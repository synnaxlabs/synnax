// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { XY } from "@synnaxlabs/x";
import { z } from "zod";

import { LookupResult } from "../../Line/core";

import { AetherContext, AetherLeaf, AetherUpdate } from "@/core/aether/worker";
import { RenderContext, RenderController } from "@/core/vis/render";

export const tooltipState = z.object({
  position: XY.z,
});

export interface TooltipProps {
  lookupX: (position: number) => Promise<LookupResult[]>;
}

export class AetherToolTip extends AetherLeaf<typeof tooltipState> {
  static readonly TYPE = "tooltip";
  static readonly stateZ = tooltipState;
  ctx: RenderContext;

  constructor(update: AetherUpdate) {
    super(update, tooltipState);
    this.ctx = RenderContext.use(update.ctx);
  }

  derive(ctx: AetherContext): void {
    RenderController.requestRender(ctx);
  }

  async render(props: TooltipProps): Promise<void> {}
}
