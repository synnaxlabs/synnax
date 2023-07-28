// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box, XY } from "@synnaxlabs/x";
import { z } from "zod";

import { AetherLeaf } from "@/core/aether/worker";
import { Color } from "@/core/color";
import { ThemeContext } from "@/core/theming/aether";
import { LookupResult } from "@/core/vis/Line/core";
import { RenderContext, RenderController } from "@/core/vis/render";

export const tooltipState = z.object({
  position: XY.z,
});

interface Derived {
  renderCtx: RenderContext;
  color: Color;
}

export interface TooltipProps {
  lookupX: (position: number) => Promise<LookupResult[]>;
  region: Box;
}

export class AetherTooltip extends AetherLeaf<typeof tooltipState, Derived> {
  static readonly TYPE = "tooltip";
  static readonly stateZ = tooltipState;
  schema = AetherTooltip.stateZ;

  derive(): Derived {
    return {
      renderCtx: RenderContext.use(this.ctx),
      color: ThemeContext.use(this.ctx).colors.gray.m1,
    };
  }

  afterUpdate(): void {
    RenderController.requestRender(this.ctx);
  }

  afterDelete(): void {
    RenderController.requestRender(this.ctx);
  }

  async render(props: TooltipProps): Promise<void> {
    if (this.deleted) return;
    const res = await props.lookupX(this.state.position.x - props.region.x);
    const { region } = props;

    const { renderCtx } = this.derived;
    const { upper2d: canvas } = renderCtx;

    canvas.strokeStyle = this.derived.color.hex;
    canvas.lineWidth = 1;

    // Take the average of the x values
    const x = res.reduce((p, c) => p + c.position.x, 0) / res.length;

    canvas.beginPath();
    canvas.moveTo(region.x + x, region.y);
    canvas.lineTo(region.x + x, region.y + props.region.height);
    canvas.stroke();

    res.forEach((r) => {
      canvas.fillStyle = r.color.setAlpha(0.5).hex;
      canvas.beginPath();
      canvas.arc(region.x + r.position.x, region.y + r.position.y, 6, 0, 2 * Math.PI);
      canvas.fill();
      // make a smaller, less transparent circle
      canvas.fillStyle = r.color.setAlpha(0.8).hex;
      canvas.beginPath();
      canvas.arc(region.x + r.position.x, region.y + r.position.y, 3, 0, 2 * Math.PI);
      canvas.fill();
    });

    const fontSize = 12;
    canvas.font = `${fontSize}px monospace`;
    canvas.fillStyle = "#ffffff";
    canvas.textAlign = "left";
    canvas.textBaseline = "top";
    const textHeight = fontSize * 1.2;
    const textWidth = 200;
    const padding = 4;
    canvas.fillText(`x: ${x.toFixed(2)}`, region.x + x + padding, region.y + padding);
    res.forEach((r, i) => {
      canvas.fillText(
        `${r.label ?? ""}: ${r.value.toFixed(2)}`,
        region.x + x + padding,
        region.y + padding + textHeight * (i + 1),
        textWidth
      );
    });
    // Draw a rectangle around the currend cursor position containing the
    // current value for each item.
  }
}
