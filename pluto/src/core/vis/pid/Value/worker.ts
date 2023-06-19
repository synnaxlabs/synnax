// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box } from "@synnaxlabs/x";
import { z } from "zod";

import { AetherContext, AetherLeaf, Update } from "@/core/aether/worker";
import { Color } from "@/core/color";
import { textDimensions } from "@/core/std/Typography/textDimensions";
import { RenderContext } from "@/core/vis/render";
import { TelemContext } from "@/core/vis/telem/TelemService";
import { PointTelemSource, pointTelemSourceMeta } from "@/core/vis/telem/TelemSource";

export const valueState = z.object({
  box: Box.z,
  telem: pointTelemSourceMeta,
  label: z.string(),
  font: z.string(),
  color: Color.z,
});

export type ValueState = z.input<typeof valueState>;
export type ParsedValueState = z.output<typeof valueState>;

export class Value extends AetherLeaf<typeof valueState> {
  private ctx: RenderContext;
  private telem: PointTelemSource;

  static readonly TYPE = "value";

  constructor(update: Update) {
    super(update, valueState);
    this.telem = TelemContext.use(update.ctx, this.state.telem.key);
    this.ctx = RenderContext.use(update.ctx);
    this.telem.onChange(() => this.render());
    this.render();
  }

  handleUpdate(ctx: AetherContext): void {
    this.telem = TelemContext.use(ctx, this.state.telem.key);
    this.ctx = RenderContext.use(ctx);
    this.telem.onChange(() => this.render());
    this.render();
  }

  render(): void {
    const box = new Box(this.state.box);
    if (box.isZero) return;
    const { canvas } = this.ctx;
    const value = this.telem.value;
    canvas.font = this.state.font;
    canvas.fillStyle = this.state.color.hex;
    const dims = textDimensions(value.toString(), this.state.font, this.ctx.canvas);
    this.ctx.erase(box);
    const pos = box.center.translate({ y: dims.height / 2, x: -dims.width / 2 });
    canvas.fillText(value.toString(), ...pos.couple);
  }
}
