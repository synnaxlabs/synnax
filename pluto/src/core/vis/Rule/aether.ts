// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box, Direction, Scale } from "@synnaxlabs/x";
import { z } from "zod";

import { RenderContext, RenderController } from "../render";

import { AetherLeaf } from "@/core/aether/worker";
import { Color } from "@/core/color";

const ruleState = z.object({
  position: z.number(),
  pixelPosition: z.number().optional().default(0),
  dragging: z.boolean(),
  color: Color.z,
});

export interface AetherRuleProps {
  direction: Direction;
  scale: Scale;
  plottingRegion: Box;
}

interface Derived {
  renderCtx: RenderContext;
}

export class AetherRule extends AetherLeaf<typeof ruleState, Derived> {
  static readonly TYPE = "Rule";
  static readonly stateZ = ruleState;

  derive(): Derived {
    return { renderCtx: RenderContext.use(this.ctx) };
  }

  afterUpdate(): void {
    RenderController.requestRender(this.ctx);
  }

  handleDelete(): void {}

  updatePositions({ scale, plottingRegion }: AetherRuleProps): number {
    if (this.state.dragging) {
      const pos = scale.pos(
        (this.state.pixelPosition - plottingRegion.top) / plottingRegion.height
      );
      this.setState((p) => ({ ...p, position: pos }));
      return this.state.pixelPosition;
    }
    const pixelPos =
      scale.reverse().pos(this.state.position) * plottingRegion.height +
      plottingRegion.top;
    this.setState((p) => ({ ...p, pixelPosition: pixelPos }));
    return pixelPos;
  }

  async render(props: AetherRuleProps): Promise<void> {
    const { renderCtx } = this.derived;
    const { direction, plottingRegion } = props;
    const { upper2d: canvas } = renderCtx;

    const pixelPos = this.updatePositions(props);
    canvas.strokeStyle = this.state.color.hex;
    canvas.lineWidth = 2;
    canvas.setLineDash([20]);
    canvas.beginPath();
    if (direction.isX) {
      canvas.moveTo(plottingRegion.left, pixelPos);
      canvas.lineTo(plottingRegion.right, pixelPos);
    } else {
      canvas.moveTo(plottingRegion.top, pixelPos);
      canvas.lineTo(plottingRegion.bottom, pixelPos);
    }
    canvas.stroke();
  }
}
