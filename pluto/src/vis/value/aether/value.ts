// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box, type Destructor, XYScale } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { color } from "@/color/core";
import { telem } from "@/telem/core";
import { noop } from "@/telem/noop";
import { dimensions } from "@/text/dimensions";
import { theming } from "@/theming/aether";
import { fontString } from "@/theming/core/fontString";
import { type PIDElement } from "@/vis/pid/aether/pid";
import { render } from "@/vis/render";

const valueState = z.object({
  box: Box.z,
  telem: telem.numericSourceSpecZ.optional().default(noop.numericSourceSpec),
  units: z.string(),
  font: z.string().optional().default(""),
  color: color.Color.z.optional().default(color.ZERO),
  precision: z.number().optional().default(2),
  width: z.number().optional().default(100),
});

export interface ValueProps {
  scale?: XYScale;
}

interface InternalState {
  render: render.Context;
  telem: telem.NumericSource;
  cleanupTelem: Destructor;
  requestRender: (() => void) | null;
  textColor: color.Color;
}

export class Value
  extends aether.Leaf<typeof valueState, InternalState>
  implements PIDElement
{
  static readonly TYPE = "value";
  static readonly z = valueState;
  schema = Value.z;

  afterUpdate(): void {
    this.internal.render = render.Context.use(this.ctx);
    const theme = theming.use(this.ctx);
    if (this.state.font.length === 0) this.state.font = fontString(theme, "p");
    if (this.state.color.isZero) this.internal.textColor = theme.colors.gray.p2;
    else this.internal.textColor = this.state.color;
    const [t, cleanupTelem] = telem.use<telem.NumericSource>(
      this.ctx,
      this.key,
      this.state.telem,
    );
    this.internal.telem = t;
    this.internal.cleanupTelem = cleanupTelem;
    this.internal.telem.onChange(() => this.requestRender());
    this.internal.requestRender = render.Controller.useOptionalRequest(this.ctx);
    this.requestRender();
  }

  afterDelete(): void {
    const { requestRender, cleanupTelem, render: renderCtx } = this.internal;
    cleanupTelem();
    if (requestRender == null) renderCtx.erase(new Box(this.state.box));
    else requestRender();
  }

  private requestRender(): void {
    const { requestRender } = this.internal;
    if (requestRender != null) requestRender();
    else void this.render({});
  }

  async render({ scale = XYScale.IDENTITY }): Promise<void> {
    const { render: renderCtx, telem } = this.internal;
    const box = new Box(this.state.box);
    if (box.isZero) return;
    const canvas = renderCtx.lower2d.applyScale(scale);

    const value = (await telem.value()).toFixed(this.state.precision);
    const valueStr = `${value} ${this.state.units}`;

    canvas.font = this.state.font;
    const dims = dimensions(valueStr, this.state.font, canvas);
    renderCtx.erase(new Box(this.prevState.box));

    if (this.state.width < dims.width)
      this.setState((p) => ({ ...p, width: dims.width }));

    const labelPosition = box.topLeft
      .translate({
        x: box.width / 2,
        y: box.height / 2,
      })
      .translate({
        y: dims.height / 2,
        x: -dims.width / 2,
      });

    canvas.fillStyle = this.internal.textColor.hex;
    canvas.fillText(valueStr, ...labelPosition.couple);
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Value.TYPE]: Value,
};
