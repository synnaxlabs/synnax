// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, location, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { theming } from "@/theming/aether";
import { Draw2D } from "@/vis/draw2d";
import { render } from "@/vis/render";
import { type ColorMap, getLUT } from "@/vis/spectrogram/aether/colormap";
import { type Core } from "@/vis/spectrogram/aether/core";

const TOOLTIP_LIST_OFFSET: xy.XY = xy.construct(12);
const TOOLTIP_LIST_SPACING = 3;
const TOOLTIP_LIST_ITEM_HEIGHT = 14;
const TOOLTIP_PADDING: xy.XY = xy.construct(6);

export const tooltipStateZ = z.object({
  cursorPosition: xy.xy.nullable().default(null),
});

export interface TooltipProps {
  plot: box.Box;
  freqMin: number;
  freqMax: number;
  minBin: number;
  maxBin: number;
  core: Core;
  fftSize: number;
  sampleRate: number;
  colorMap: ColorMap;
  dbMin: number;
  dbMax: number;
}

interface InternalState {
  renderCtx: render.Context;
  draw: Draw2D;
  theme: theming.Theme;
}

export class Tooltip extends aether.Leaf<typeof tooltipStateZ, InternalState> {
  static readonly TYPE = "spectrogram-tooltip";
  schema = tooltipStateZ;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.renderCtx = render.Context.use(ctx);
    i.theme = theming.use(ctx);
    i.draw = new Draw2D(i.renderCtx.upper2d, i.theme);
    render.request(ctx, "tool");
  }

  afterDelete(ctx: aether.Context): void {
    render.request(ctx, "tool");
  }

  render(props: TooltipProps): void {
    const { cursorPosition } = this.state;
    if (cursorPosition == null) return;
    if (!box.contains(props.plot, cursorPosition)) return;

    const { internal: i } = this;
    const { draw } = i;
    const { plot, freqMin, freqMax, minBin, maxBin, core } = props;
    const { fftSize, sampleRate, colorMap, dbMin, dbMax } = props;
    const ruleColor = i.theme.colors.gray.l7;

    draw.rule({
      stroke: ruleColor,
      lineWidth: 1,
      lineDash: 0,
      direction: "y",
      region: plot,
      position: cursorPosition.x,
    });
    draw.rule({
      stroke: ruleColor,
      lineWidth: 1,
      lineDash: 0,
      direction: "x",
      region: plot,
      position: cursorPosition.y,
    });

    const plotLeft = box.left(plot);
    const plotTop = box.top(plot);
    const plotWidth = box.width(plot);
    const plotHeight = box.height(plot);
    const yFrac = (cursorPosition.y - plotTop) / plotHeight;
    const freq = freqMin + (1 - yFrac) * (freqMax - freqMin);

    const halfFFT = fftSize / 2;
    const binFreqStep = sampleRate / fftSize;
    const bin = Math.round(freq / binFreqStep);
    const canvasX = Math.round(
      ((cursorPosition.x - plotLeft) / plotWidth) * (core.width - 1),
    );
    const pixelY = halfFFT - 1 - Math.max(minBin, Math.min(maxBin - 1, bin));

    let db = NaN;
    if (
      canvasX >= 0 &&
      canvasX < core.width &&
      pixelY >= 0 &&
      pixelY < halfFFT
    ) {
      const pixel = core.offscreenCtx.getImageData(canvasX, pixelY, 1, 1).data;
      const lut = getLUT(colorMap);
      let bestDist = Infinity;
      let bestIdx = 0;
      for (let j = 0; j < 256; j++) {
        const li = j << 2;
        const dr = pixel[0] - lut[li];
        const dg = pixel[1] - lut[li + 1];
        const dbl = pixel[2] - lut[li + 2];
        const dist = dr * dr + dg * dg + dbl * dbl;
        if (dist < bestDist) {
          bestDist = dist;
          bestIdx = j;
        }
      }
      db = dbMin + (bestIdx / 255) * (dbMax - dbMin);
    }

    const freqLabel =
      freq >= 1000 ? `${(freq / 1000).toFixed(2)} kHz` : `${Math.round(freq)} Hz`;
    const dbLabel = isNaN(db) ? "\u2014" : `${db.toFixed(1)} dB`;

    const items = [
      { label: "Freq", value: freqLabel },
      { label: "Power", value: dbLabel },
    ];

    const xFrac = (cursorPosition.x - plotLeft) / plotWidth;
    const root = { ...location.TOP_LEFT };
    if (xFrac > 0.6) root.x = "right";
    if (yFrac > 0.6) root.y = "bottom";

    const maxLabelLen = Math.max(
      ...items.map((it) => it.label.length + it.value.length),
    );

    draw.list({
      root,
      offset: TOOLTIP_LIST_OFFSET,
      length: items.length,
      padding: TOOLTIP_PADDING,
      itemHeight: TOOLTIP_LIST_ITEM_HEIGHT,
      spacing: TOOLTIP_LIST_SPACING,
      width: maxLabelLen * 7 + 48,
      position: cursorPosition,
      draw: (idx, b) => {
        const item = items[idx];
        draw.text({
          position: box.topLeft(b),
          text: item.label,
          level: "small",
          weight: 500,
        });
        draw.text({
          position: xy.translateY(box.topRight(b), -1),
          text: item.value,
          level: "small",
          justify: "right",
          code: true,
          shade: 10,
        });
      },
    });
  }
}
