// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, direction, location, type scale, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { color } from "@/color/core";
import { theming } from "@/theming/aether";
import { Draw2D } from "@/vis/draw2d";
import { render } from "@/vis/render";

export const ruleStateZ = z.object({
  position: z.number().optional(),
  pixelPosition: z.number().optional(),
  dragging: z.boolean(),
  lineWidth: z.number().optional().default(1),
  lineDash: z.number().optional().default(20),
  color: color.Color.z,
});

export interface RuleProps {
  location: location.Location;
  decimalToDataScale: scale.Scale;
  plot: box.Box;
  container: box.Box;
}

interface InternalState {
  renderCtx: render.Context;
  draw: Draw2D;
}

export class Rule extends aether.Leaf<typeof ruleStateZ, InternalState> {
  static readonly TYPE = "Rule";

  schema = ruleStateZ;

  async afterUpdate(): Promise<void> {
    this.internal.renderCtx = render.Context.use(this.ctx);
    const theme = theming.use(this.ctx);
    this.internal.draw = new Draw2D(this.internal.renderCtx.upper2d, theme);
    render.Controller.requestRender(this.ctx, render.REASON_TOOL);
  }

  async afterDelete(): Promise<void> {
    render.Controller.requestRender(this.ctx, render.REASON_TOOL);
  }

  updateXPositions({ decimalToDataScale, plot, container }: RuleProps): number {
    if (this.state.dragging && this.state.pixelPosition != null) {
      const pos = decimalToDataScale.pos(
        (this.state.pixelPosition - box.left(plot) + box.left(container)) /
          box.width(plot),
      );
      this.setState((p) => ({ ...p, position: pos }));
      return this.state.pixelPosition;
    }
    if (this.state.position == null) {
      // Calculate the position of the rule at the middle of the plot
      const pos = decimalToDataScale.pos(0.5);
      this.setState((p) => ({ ...p, position: pos }));
    }
    const pixelPos =
      decimalToDataScale.reverse().pos(this.state.position as number) *
        box.width(plot) +
      box.left(plot) -
      box.left(container);
    if (!isNaN(pixelPos)) this.setState((p) => ({ ...p, pixelPosition: pixelPos }));
    return pixelPos;
  }

  updateYPositions({ decimalToDataScale: scale, plot, container }: RuleProps): number {
    if (this.state.dragging && this.state.pixelPosition != null) {
      const pos = scale.pos(
        (this.state.pixelPosition - box.top(plot) + box.top(container)) /
          box.height(plot),
      );
      this.setState((p) => ({ ...p, position: pos }));
      return this.state.pixelPosition;
    }
    if (this.state.position == null) {
      // Calculate the position of the rule at the middle of the plot
      const pos = scale.pos(0.5);
      this.setState((p) => ({ ...p, position: pos }));
    }
    const pixelPos =
      scale.reverse().pos(this.state.position as number) * box.height(plot) +
      box.top(plot) -
      box.top(container);
    if (!isNaN(pixelPos)) this.setState((p) => ({ ...p, pixelPosition: pixelPos }));
    return pixelPos;
  }

  private updatePosition(props: RuleProps): number {
    const dir = direction.construct(props.location);
    if (dir === "y") return this.updateXPositions(props);
    return this.updateYPositions(props);
  }

  async render(props: RuleProps): Promise<void> {
    if (this.deleted) return;
    const { renderCtx } = this.internal;
    const { location: l, plot: plottingRegion } = props;
    const direction = location.direction(l);
    const { upper2d: canvas } = renderCtx;
    const draw = this.internal.draw;

    let pixelPos = this.updatePosition(props);
    if (direction == "x") pixelPos += box.top(props.container);
    else if (direction == "y") pixelPos += box.left(props.container);

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
    if (l === "left") {
      canvas.moveTo(box.left(plottingRegion), pixelPos);
      canvas.lineTo(box.left(plottingRegion) - 5, pixelPos - 5);
      canvas.lineTo(box.left(plottingRegion) - 5, pixelPos + 5);
    } else if (l === "right") {
      canvas.moveTo(box.right(plottingRegion), pixelPos);
      canvas.lineTo(box.right(plottingRegion) + 5, pixelPos - 5);
      canvas.lineTo(box.right(plottingRegion) + 5, pixelPos + 5);
    } else if (l === "top") {
      canvas.moveTo(pixelPos, box.top(plottingRegion));
      canvas.lineTo(pixelPos - 5, box.top(plottingRegion) - 5);
      canvas.lineTo(pixelPos + 5, box.top(plottingRegion) - 5);
    } else {
      console.log("COND");
      canvas.moveTo(pixelPos, box.bottom(plottingRegion) - 2);
      canvas.lineTo(pixelPos - 5, box.bottom(plottingRegion) + 5);
      canvas.lineTo(pixelPos + 5, box.bottom(plottingRegion) + 5);
    }
    canvas.closePath();
    canvas.fill();
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Rule.TYPE]: Rule,
};
