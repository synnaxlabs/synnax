// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, color, scale, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { theming } from "@/theming/aether";
import { fontString } from "@/theming/base/fontString";
import { axis } from "@/vis/axis";
import { type FillTextOptions } from "@/vis/draw2d/canvas";
import { render } from "@/vis/render";
import { type ColorMap, getLUT } from "@/vis/spectrogram/aether/colormap";

const COLOR_BAR_WIDTH = 12;
const COLOR_BAR_GAP = 8;
const TICK_PADDING = 6;
const GRADIENT_STOPS = 32;
const FILL_TEXT_OPTIONS: FillTextOptions = { useAtlas: true };

const colorBarStateZ = z.object({});

export interface ColorBarProps {
  plot: box.Box;
  colorMap: ColorMap;
  dbMin: number;
  dbMax: number;
  currentSize: number;
}

export interface ColorBarRenderResult {
  size: number;
}

interface InternalState {
  renderCtx: render.Context;
  theme: theming.Theme;
  axis: axis.Axis;
  size: number;
}

export class ColorBar extends aether.Leaf<typeof colorBarStateZ, InternalState> {
  static readonly TYPE = "spectrogram-color-bar";
  schema = colorBarStateZ;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.renderCtx = render.Context.use(ctx);
    i.theme = theming.use(ctx);
    const axisColor = i.theme.colors.gray.l10;
    const gridColor = i.theme.colors.gray.l1;
    const font = fontString(i.theme, { level: "small", code: true });
    const state = axis.axisStateZ.parse({
      color: axisColor,
      gridColor,
      font,
      type: "linear",
      showGrid: false,
      location: "right",
    });
    i.axis = axis.newCanvas("right", i.renderCtx, state);
    i.size ??= 60;
  }

  render(props: ColorBarProps): ColorBarRenderResult {
    const { internal: i } = this;
    const { plot, colorMap, dbMin, dbMax } = props;
    const canvas = i.renderCtx.lower2d;

    const plotRight = box.left(plot) + box.width(plot);
    const plotTop = box.top(plot);
    const plotHeight = box.height(plot);
    const barLeft = plotRight + COLOR_BAR_GAP;
    const lut = getLUT(colorMap);

    const grad = canvas.createLinearGradient(0, plotTop, 0, plotTop + plotHeight);
    for (let j = 0; j <= GRADIENT_STOPS; j++) {
      const frac = j / GRADIENT_STOPS;
      const lutIdx = Math.max(0, Math.min(255, Math.round((1 - frac) * 255))) * 4;
      grad.addColorStop(
        frac,
        `rgb(${lut[lutIdx]},${lut[lutIdx + 1]},${lut[lutIdx + 2]})`,
      );
    }
    canvas.fillStyle = grad;
    canvas.fillRect(barLeft, plotTop, COLOR_BAR_WIDTH, plotHeight);

    const axisColor = color.hex(i.theme.colors.gray.l10);
    canvas.strokeStyle = axisColor;
    canvas.strokeRect(barLeft, plotTop, COLOR_BAR_WIDTH, plotHeight);

    const decimalToDataScale = scale.Scale.scale(dbMax, dbMin);
    const pos = xy.construct(barLeft + COLOR_BAR_WIDTH, plotTop);
    const { size } = i.axis.render({
      plot,
      position: pos,
      size: 0,
      decimalToDataScale,
    });

    const font = fontString(i.theme, { level: "small", code: true });
    canvas.font = font;
    canvas.fillStyle = axisColor;
    const dbLabel = "dB";
    const dbD = canvas.textDimensions(dbLabel, FILL_TEXT_OPTIONS);
    canvas.fillText(
      dbLabel,
      barLeft + (COLOR_BAR_WIDTH - dbD.width) / 2,
      plotTop - TICK_PADDING,
      undefined,
      FILL_TEXT_OPTIONS,
    );

    const totalSize = COLOR_BAR_GAP + COLOR_BAR_WIDTH + size + TICK_PADDING;
    i.size = totalSize;
    return { size: totalSize };
  }
}
