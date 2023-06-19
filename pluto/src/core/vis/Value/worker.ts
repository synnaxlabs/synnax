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

import { AetherContext, AetherLeaf, Update } from "@/core/aether/worker";
import { Color } from "@/core/color";
import { textDimensions } from "@/core/std/Typography/textDimensions";
import { RenderContext, RenderController } from "@/core/vis/render";
import { TelemContext } from "@/core/vis/telem/TelemService";
import { PointTelemSource, pointTelemSourceMeta } from "@/core/vis/telem/TelemSource";

export const valueState = z.object({
  box: Box.z,
  telem: pointTelemSourceMeta,
  units: z.string(),
  font: z.string(),
  color: Color.z,
  position: XY.z.optional(),
});

export type ValueState = z.input<typeof valueState>;

export interface ValueProps {
  position: XY;
}

export class Value extends AetherLeaf<typeof valueState> {
  private renderCtx: RenderContext;
  private telem: PointTelemSource;
  private _requestRender: (() => void) | null;

  static readonly TYPE = "value";

  constructor(update: Update) {
    super(update, valueState);
    this.telem = TelemContext.use(update.ctx, this.state.telem.key);
    this.renderCtx = RenderContext.use(update.ctx);
    this._requestRender = RenderController.useOptionalRequest(update.ctx);
    this.telem.onChange(() => this.render());
  }

  handleUpdate(ctx: AetherContext): void {
    this.telem = TelemContext.use(ctx, this.state.telem.key);
    this.renderCtx = RenderContext.use(ctx);
    this._requestRender = RenderController.useOptionalRequest(ctx);
    this.requestRender();
    this.telem.onChange(() => this.render());
  }

  handleDelete(): void {
    this.requestRender();
  }

  private requestRender(): void {
    if (this._requestRender != null) this._requestRender();
    else this.render();
  }

  render(props: ValueProps = { position: XY.ZERO }): void {
    const box = new Box(this.state.box);
    if (box.isZero) return;
    const { canvas } = this.renderCtx;
    const value = `${this.telem.value} ${this.state.units}`;
    const statePos = this.state.position ?? XY.ZERO;
    canvas.font = this.state.font;
    const dims = textDimensions(value, this.state.font, this.renderCtx.canvas);
    this.renderCtx.erase(box);

    canvas.fillStyle = new Color("#fc3d03").setAlpha(1).hex;
    canvas.beginPath();
    const startPos = statePos.translate(props.position).translateY(28);
    canvas.roundRect(...startPos.couple, box.width, box.height, 2);
    canvas.fill();

    const pos = startPos
      .translate({
        x: box.width / 2,
        y: box.height / 2,
      })
      .translate({ y: dims.height / 2, x: -dims.width / 2 });
    canvas.fillStyle = this.state.color.hex;
    canvas.fillText(value.toString(), ...pos.couple);
  }
}
