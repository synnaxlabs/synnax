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
import { PIDElement } from "@/core/vis/PID/aether";
import { RenderContext, RenderController } from "@/core/vis/render";
import { TelemContext } from "@/core/vis/telem/TelemContext";
import {
  NumericTelemSource,
  numericTelemSourceProps,
} from "@/core/vis/telem/TelemSource";

const valueState = z.object({
  box: Box.z,
  telem: numericTelemSourceProps,
  units: z.string(),
  font: z.string(),
  color: Color.z,
  precision: z.number().optional().default(2),
  width: z.number().optional().default(100),
});

export interface ValueProps {
  position: XY;
}

export class AetherValue extends AetherLeaf<typeof valueState> implements PIDElement {
  private renderCtx: RenderContext;
  private telem: NumericTelemSource;
  private releaseTelem?: () => void;
  private _requestRender: (() => void) | null;

  static readonly TYPE = "value";
  static readonly stateZ = valueState;

  constructor(update: Update) {
    super(update, valueState);
    [this.telem, this.releaseTelem] = TelemContext.use(
      update.ctx,
      this.key,
      this.state.telem
    );
    this.renderCtx = RenderContext.use(update.ctx);
    this._requestRender = RenderController.useOptionalRequest(update.ctx);
    this.telem.onChange(() => this.requestRender());
    this.requestRender();
  }

  handleUpdate(ctx: AetherContext): void {
    [this.telem, this.releaseTelem] = TelemContext.use(ctx, this.key, this.state.telem);
    this.renderCtx = RenderContext.use(ctx);
    this._requestRender = RenderController.useOptionalRequest(ctx);
    this.telem.onChange(() => this.requestRender());
    this.requestRender();
  }

  handleDelete(): void {
    this.releaseTelem?.();
    if (this._requestRender == null) this.renderCtx.erase(new Box(this.state.box));
    else this._requestRender();
  }

  private requestRender(): void {
    if (this._requestRender != null) this._requestRender();
    else void this.render();
  }

  async render(props?: ValueProps): Promise<void> {
    const box = new Box(this.state.box);
    if (box.isZero) return;
    const { lower2d: canvas } = this.renderCtx;

    const value = (await this.telem.value()).toFixed(this.state.precision);
    const valueStr = `${value} ${this.state.units}`;

    canvas.font = this.state.font;
    const dims = textDimensions(valueStr, this.state.font, canvas);
    this.renderCtx.erase(new Box(this.prevState.box));

    if (this.state.width < dims.width)
      this.setState((p) => ({ ...p, width: dims.width }));

    const labelPosition = box.topLeft
      .translate(props?.position ?? XY.ZERO)
      .translate({
        x: box.width / 2,
        y: box.height / 2,
      })
      .translate({ y: dims.height / 2, x: -dims.width / 2 });

    canvas.fillStyle = this.state.color.hex;
    canvas.fillText(valueStr, ...labelPosition.couple);
  }
}
