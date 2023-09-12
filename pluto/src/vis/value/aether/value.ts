// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Destructor, box, scale, xy } from "@synnaxlabs/x";
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
  box: box.box,
  telem: telem.numericSourceSpecZ.optional().default(noop.numericSourceSpec),
  units: z.string(),
  font: z.string().optional().default(""),
  color: color.Color.z.optional().default(color.ZERO),
  precision: z.number().optional().default(2),
  width: z.number().optional().default(100),
});

export interface ValueProps {
  scale?: scale.XY;
}

interface InternalState {
  render: render.Context;
  telem: telem.NumericSource;
  cleanupTelem: Destructor;
  requestRender: render.RequestF | null;
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
    if (requestRender == null)
      renderCtx.erase(box.construct(this.state.box), xy.ZERO, "lower2d");
    else requestRender(render.REASON_LAYOUT);
  }

  private requestRender(): void {
    const { requestRender } = this.internal;
    if (requestRender != null) requestRender(render.REASON_LAYOUT);
    else void this.render({});
  }

  async render({ s = scale.XY.IDENTITY }): Promise<void> {
    const { render: renderCtx, telem } = this.internal;
    const b = box.construct(this.state.box);
    if (box.isZero(b)) return;
    const canvas = renderCtx.lower2d.applyScale(s);

    const value = (await telem.value()).toFixed(this.state.precision);
    const valueStr = `${value} ${this.state.units}`;

    canvas.font = this.state.font;
    const dims = dimensions(valueStr, this.state.font, canvas);
    renderCtx.erase(box.construct(this.prevState.box));

    if (this.state.width < dims.width)
      this.setState((p) => ({ ...p, width: dims.width }));

    const labelPosition = xy.couple(
      xy.translate(
        box.topLeft(b),
        {
          x: box.width(b) / 2,
          y: box.height(b) / 2,
        },
        {
          y: dims.height / 2,
          x: -dims.width / 2,
        },
      ),
    );

    canvas.fillStyle = this.internal.textColor.hex;
    canvas.fillText(valueStr, ...labelPosition);
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Value.TYPE]: Value,
};
