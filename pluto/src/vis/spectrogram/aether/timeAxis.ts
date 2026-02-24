// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, scale, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { theming } from "@/theming/aether";
import { fontString } from "@/theming/base/fontString";
import { axis } from "@/vis/axis";
import { render } from "@/vis/render";

const timeAxisStateZ = z.object({});

export interface TimeAxisProps {
  plot: box.Box;
  totalSeconds: number;
  currentSize: number;
}

interface InternalState {
  renderCtx: render.Context;
  axis: axis.Axis;
}

export class TimeAxis extends aether.Leaf<typeof timeAxisStateZ, InternalState> {
  static readonly TYPE = "spectrogram-time-axis";
  schema = timeAxisStateZ;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.renderCtx = render.Context.use(ctx);
    const theme = theming.use(ctx);
    const axisColor = theme.colors.gray.l10;
    const gridColor = theme.colors.gray.l1;
    const font = fontString(theme, { level: "small", code: true });
    const state = axis.axisStateZ.parse({
      color: axisColor,
      gridColor,
      font,
      type: "linear",
      showGrid: false,
      location: "bottom",
    });
    i.axis = axis.newCanvas("bottom", i.renderCtx, state);
  }

  render(props: TimeAxisProps): axis.RenderResult {
    const { internal: i } = this;
    const { plot, totalSeconds } = props;
    const decimalToDataScale = scale.Scale.scale(-totalSeconds, 0);
    const pos = xy.construct(box.left(plot), box.top(plot) + box.height(plot));
    return i.axis.render({
      plot,
      position: pos,
      size: 0,
      decimalToDataScale,
    });
  }
}
