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

const freqAxisStateZ = z.object({});

export interface FreqAxisProps {
  plot: box.Box;
  freqMin: number;
  freqMax: number;
  currentSize: number;
}

export interface FreqAxisRenderResult {
  size: number;
}

interface InternalState {
  renderCtx: render.Context;
  axis: axis.Axis;
  size: number;
}

export class FreqAxis extends aether.Leaf<typeof freqAxisStateZ, InternalState> {
  static readonly TYPE = "spectrogram-freq-axis";
  schema = freqAxisStateZ;

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
      location: "left",
    });
    i.axis = axis.newCanvas("left", i.renderCtx, state);
    i.size ??= 50;
  }

  render(props: FreqAxisProps): FreqAxisRenderResult {
    const { internal: i } = this;
    const { plot, freqMin, freqMax, currentSize } = props;
    const decimalToDataScale = scale.Scale.scale(freqMax, freqMin);
    const pos = xy.construct(box.left(plot), box.top(plot));
    const { size } = i.axis.render({
      plot,
      position: pos,
      size: currentSize,
      decimalToDataScale,
    });
    i.size = size;
    return { size };
  }
}
