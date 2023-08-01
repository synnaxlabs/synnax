// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box, Location, Scale } from "@synnaxlabs/x";
import { z } from "zod";

import { Draw2D } from "../draw2d";
import { RenderContext, RenderController } from "../render";

import { AetherLeaf } from "@/core/aether/worker";
import { Color } from "@/core/color";
import { ThemeContext } from "@/core/theming/aether";

const ruleState = z.object({
  position: z.number(),
  pixelPosition: z.number().optional().default(0),
  dragging: z.boolean(),
  lineWidth: z.number().optional().default(1),
  lineDash: z.number().optional().default(20),
  color: Color.z,
});

export interface AetherRuleProps {
  location: Location;
  decimalToDataScale: Scale;
  plot: Box;
  container: Box;
}

interface InternalState {
  renderCtx: RenderContext;
  draw: Draw2D;
}

export class AetherRule extends AetherLeaf<typeof ruleState, InternalState> {
  static readonly TYPE = "Rule";
  static readonly stateZ = ruleState;

  schema = AetherRule.stateZ;

  afterUpdate(): void {
    this.internal.renderCtx = RenderContext.use(this.ctx);
    const theme = ThemeContext.use(this.ctx);
    this.internal.draw = new Draw2D(this.internal.renderCtx.upper2d, theme);
    RenderController.requestRender(this.ctx);
  }

  updatePositions({
    decimalToDataScale: scale,
    plot,
    container,
  }: AetherRuleProps): number {
    if (this.state.dragging) {
      const pos = scale.pos(
        (this.state.pixelPosition - plot.top + container.top) / plot.height
      );
      this.setState((p) => ({ ...p, position: pos }));
      return this.state.pixelPosition;
    }
    const pixelPos =
      scale.reverse().pos(this.state.position) * plot.height + plot.top - container.top;
    if (!isNaN(pixelPos)) this.setState((p) => ({ ...p, pixelPosition: pixelPos }));
    return pixelPos;
  }

  async render(props: AetherRuleProps): Promise<void> {
    const { renderCtx } = this.internal;
    const { location, plot: plottingRegion } = props;
    const direction = location.direction;
    const { upper2d: canvas } = renderCtx;
    const draw = this.internal.draw;

    let pixelPos = this.updatePositions(props);
    pixelPos += props.container.top;

    draw.rule({
      stroke: this.state.color,
      lineWidth: this.state.lineWidth,
      lineDash: this.state.lineDash,
      direction,
      region: plottingRegion,
      position: pixelPos,
    });

    canvas.fillStyle = this.state.color.hex;
    canvas.beginPath();
    if (location.isLeft) {
      canvas.moveTo(plottingRegion.left, pixelPos);
      canvas.lineTo(plottingRegion.left - 5, pixelPos - 5);
      canvas.lineTo(plottingRegion.left - 5, pixelPos + 5);
    } else if (location.isRight) {
      canvas.moveTo(plottingRegion.right, pixelPos);
      canvas.lineTo(plottingRegion.right + 5, pixelPos - 5);
      canvas.lineTo(plottingRegion.right + 5, pixelPos + 5);
    }
    canvas.closePath();
    canvas.fill();
  }
}
