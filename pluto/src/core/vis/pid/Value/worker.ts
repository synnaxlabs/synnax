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

import { RenderContext } from "../../render";
import { TelemProvider } from "../../telem/TelemService";
import { PointTelemSource, pointTelemSourceMeta } from "../../telem/TelemSource";

import { Color } from "@/core/color";
import { textDimensions } from "@/core/std/Typography/textDimensions";
import { VLeaf } from "@/core/virtual/worker";

export const valueState = z.object({
  position: XY.z,
  telem: pointTelemSourceMeta,
  label: z.string(),
  font: z.string(),
  color: Color.z,
});

export type ValueState = z.input<typeof valueState>;
export type ParsedValueState = z.output<typeof valueState>;

export interface ValueContext {
  position: XY;
}

export class Value extends VLeaf<ValueState, ParsedValueState> {
  private readonly ctx: RenderContext;
  private readonly telem: PointTelemSource;
  private readonly requestRender: () => void;

  static readonly TYPE = "value";

  constructor(
    key: string,
    ctx: RenderContext,
    state: any,
    telemProv: TelemProvider,
    requestRender: () => void
  ) {
    super(Value.TYPE, key, state, valueState);
    this.telem = telemProv.use(state.telem.key);
    this.ctx = ctx;
    this.requestRender = requestRender;
    this.bindStateHook(() => this.requestRender());
  }

  render(ctx: ValueContext): void {
    const value = this.telem.value();
    this.ctx.canvas.font = this.state.font;
    this.ctx.canvas.fillStyle = this.state.color.hex;
    const labelDims = textDimensions(
      this.state.label.toString(),
      this.ctx.canvas.font,
      this.ctx.canvas
    );
    this.ctx.canvas.fillText(
      value.toString(),
      ...this.state.position.translate(ctx.position).translate({
        y: labelDims.height * 4,
        x: labelDims.width / 2,
      }).couple
    );
  }
}
