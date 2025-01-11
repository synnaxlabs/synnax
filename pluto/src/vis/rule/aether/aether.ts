// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, location, type scale } from "@synnaxlabs/x";
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

const PIXEL_POS_UPDATE_DIST = 3;

export class Rule extends aether.Leaf<typeof ruleStateZ, InternalState> {
  static readonly TYPE = "Rule";

  schema = ruleStateZ;
  lastUpdateRef: number | null = null;

  async afterUpdate(): Promise<void> {
    this.internal.renderCtx = render.Context.use(this.ctx);
    const theme = theming.use(this.ctx);
    this.internal.draw = new Draw2D(this.internal.renderCtx.upper2d, theme);
    render.Controller.requestRender(this.ctx, render.REASON_TOOL);
  }

  async afterDelete(): Promise<void> {
    render.Controller.requestRender(this.ctx, render.REASON_TOOL);
  }

  updatePositions({ decimalToDataScale: scale, plot }: RuleProps): number {
    const isDragging = this.state.dragging;
    const wasDragging = this.prevState.dragging && !isDragging;

    if ((isDragging || wasDragging) && this.state.pixelPosition != null) {
      this.lastUpdateRef ??= this.state.pixelPosition;
      const delta = Math.abs(this.state.pixelPosition - this.lastUpdateRef);
      if (delta < PIXEL_POS_UPDATE_DIST && !wasDragging)
        return this.state.pixelPosition;
      this.lastUpdateRef = this.state.pixelPosition;
      const pos = scale.pos(this.state.pixelPosition / box.height(plot));
      this.setState((p) => ({ ...p, position: pos }));
      return this.state.pixelPosition;
    }

    if (this.state.position == null) {
      // Calculate the position of the rule at the middle of the plot
      const pos = scale.pos(0.5);
      this.setState((p) => ({ ...p, position: pos }));
    }

    const pixelPos =
      scale.reverse().pos(this.state.position as number) * box.height(plot);
    if (!isNaN(pixelPos)) {
      if (this.state.pixelPosition != null) {
        const delta = Math.abs(pixelPos - this.state.pixelPosition);
        if (delta < 1) return this.state.pixelPosition;
      }
      this.setState((p) => ({ ...p, pixelPosition: pixelPos }));
    }
    return pixelPos;
  }

  async render(props: RuleProps): Promise<void> {
    if (this.deleted) return;
    const { renderCtx } = this.internal;
    const { location: l, plot: plottingRegion } = props;
    const direction = location.direction(l);
    const { upper2d: canvas } = renderCtx;
    const draw = this.internal.draw;

    // The pixel position we calculate for the main thread is relative
    // to the plot box, so we need to offset it to match the pixel positions
    // of the canvas.
    const pos = this.updatePositions(props) + box.top(props.plot);

    draw.rule({
      stroke: this.state.color,
      lineWidth: this.state.lineWidth,
      lineDash: this.state.lineDash,
      direction,
      region: plottingRegion,
      position: pos,
    });

    canvas.fillStyle = this.state.color.hex;
    canvas.lineJoin = "round";
    canvas.lineWidth = 3.5;
    canvas.lineCap = "round";
    canvas.beginPath();
    const TRIANGLE_SIZE = 4;
    if (l === "left") {
      const arrowPos = box.left(plottingRegion) - 1;
      canvas.moveTo(arrowPos, pos);
      canvas.lineTo(arrowPos - TRIANGLE_SIZE, pos - TRIANGLE_SIZE);
      canvas.lineTo(arrowPos - TRIANGLE_SIZE, pos + TRIANGLE_SIZE);
    } else if (l === "right") {
      canvas.moveTo(box.right(plottingRegion), pos);
      canvas.lineTo(box.right(plottingRegion) + TRIANGLE_SIZE, pos - TRIANGLE_SIZE);
      canvas.lineTo(box.right(plottingRegion) + TRIANGLE_SIZE, pos + TRIANGLE_SIZE);
    }
    canvas.closePath();
    canvas.stroke();
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Rule.TYPE]: Rule,
};
