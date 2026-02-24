// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { telem } from "@/telem/aether";
import { render } from "@/vis/render";
import { ColorBar } from "@/vis/spectrogram/aether/colorBar";
import { Core, coreStateZ } from "@/vis/spectrogram/aether/core";
import { FreqAxis } from "@/vis/spectrogram/aether/freqAxis";
import { TimeAxis } from "@/vis/spectrogram/aether/timeAxis";
import { Tooltip } from "@/vis/spectrogram/aether/tooltip";

const TOP_PADDING = 20;

const spectrogramStateZ = coreStateZ.extend({
  box: box.box,
  telem: telem.seriesSourceSpecZ.default(telem.noopSeriesSourceSpec),
  visible: z.boolean().default(true),
});

type Children = FreqAxis | TimeAxis | ColorBar | Tooltip;

const CANVAS_VARIANTS: render.Canvas2DVariant[] = ["upper2d", "lower2d"];

interface InternalState {
  renderCtx: render.Context;
  telem: telem.SeriesSource;
  stopListening?: () => void;
  core: Core;
  leftAxisSize: number;
  bottomAxisSize: number;
  rightAxisSize: number;
}

export class Spectrogram extends aether.Composite<
  typeof spectrogramStateZ,
  InternalState,
  Children
> {
  static readonly TYPE = "spectrogram";
  static readonly z = spectrogramStateZ;
  schema = Spectrogram.z;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.renderCtx = render.Context.use(ctx);

    i.leftAxisSize ??= 50;
    i.bottomAxisSize ??= 24;
    i.rightAxisSize ??= 60;

    const coreState = coreStateZ.parse(this.state);
    const plotRegion = this.plotRegion();
    const width = Math.max(1, Math.round(box.width(plotRegion)));

    if (i.core == null) i.core = new Core(coreState, width);
    else i.core.updateState(coreState, width);

    i.telem = telem.useSource(ctx, this.state.telem, i.telem);
    i.stopListening?.();
    i.stopListening = i.telem.onChange(() => this.requestRender());
    this.requestRender();
  }

  afterDelete(): void {
    const { internal: i } = this;
    i.stopListening?.();
    i.telem.cleanup?.();
    this.requestRender();
  }

  private requestRender(): void {
    this.internal.renderCtx.loop.set({
      key: `${this.type}-${this.key}`,
      render: () => this.render(),
      priority: "high",
      canvases: CANVAS_VARIANTS,
    });
  }

  private plotRegion(): box.Box {
    const b = this.state.box;
    const { leftAxisSize, bottomAxisSize, rightAxisSize } = this.internal;
    if (box.areaIsZero(b)) return b;
    return box.construct(
      { x: box.left(b) + leftAxisSize, y: box.top(b) + TOP_PADDING },
      {
        width: Math.max(1, box.width(b) - leftAxisSize - rightAxisSize),
        height: Math.max(1, box.height(b) - TOP_PADDING - bottomAxisSize),
      },
    );
  }

  private get freqAxes(): readonly FreqAxis[] {
    return this.childrenOfType<FreqAxis>(FreqAxis.TYPE);
  }

  private get timeAxes(): readonly TimeAxis[] {
    return this.childrenOfType<TimeAxis>(TimeAxis.TYPE);
  }

  private get colorBars(): readonly ColorBar[] {
    return this.childrenOfType<ColorBar>(ColorBar.TYPE);
  }

  private get tooltips(): readonly Tooltip[] {
    return this.childrenOfType<Tooltip>(Tooltip.TYPE);
  }

  private render(): render.Cleanup | undefined {
    const { internal: i } = this;
    const { box: b, visible } = this.state;
    if (box.areaIsZero(b) || !visible) {
      const eraseRegion = box.copy(b);
      return ({ canvases }) =>
        i.renderCtx.erase(eraseRegion, xy.ZERO, ...canvases);
    }

    const [, multiSeries] = i.telem.value();
    i.core.processTelemetry(multiSeries);

    const { sampleRate, fftSize, colorMap, dbMin, dbMax } = this.state;
    const freqMin = this.state.freqMin;
    const freqMax = this.state.freqMax > 0 ? this.state.freqMax : sampleRate / 2;
    const halfFFT = fftSize / 2;
    const binFreqStep = sampleRate / fftSize;
    const minBin = Math.max(0, Math.floor(freqMin / binFreqStep));
    const maxBin = Math.min(halfFFT, Math.ceil(freqMax / binFreqStep));

    const plotRegion = this.plotRegion();

    const clearCanvasScissor = i.renderCtx.scissor(
      box.construct(b),
      xy.ZERO,
      CANVAS_VARIANTS,
    );

    const clearPlotScissor = i.renderCtx.scissor(plotRegion, xy.ZERO, ["lower2d"]);
    i.renderCtx.lower2d.drawImage(
      i.core.offscreenCanvas,
      0,
      halfFFT - maxBin,
      i.core.width,
      maxBin - minBin,
      box.left(plotRegion),
      box.top(plotRegion),
      box.width(plotRegion),
      box.height(plotRegion),
    );
    clearPlotScissor();

    this.freqAxes.forEach((a) => {
      const { size } = a.render({
        plot: plotRegion,
        freqMin,
        freqMax,
        currentSize: i.leftAxisSize,
      });
      i.leftAxisSize = size;
    });

    const totalSeconds = (i.core.width * i.core.hopSize) / sampleRate;
    this.timeAxes.forEach((a) => {
      const { size } = a.render({
        plot: plotRegion,
        totalSeconds,
        currentSize: i.bottomAxisSize,
      });
      i.bottomAxisSize = size;
    });

    this.colorBars.forEach((cb) => {
      const { size } = cb.render({
        plot: plotRegion,
        colorMap,
        dbMin,
        dbMax,
        currentSize: i.rightAxisSize,
      });
      i.rightAxisSize = size;
    });

    this.tooltips.forEach((t) =>
      t.render({
        plot: plotRegion,
        freqMin,
        freqMax,
        minBin,
        maxBin,
        core: i.core,
        fftSize,
        sampleRate,
        colorMap,
        dbMin,
        dbMax,
      }),
    );

    clearCanvasScissor();

    const eraseRegion = box.copy(b);
    return ({ canvases }) =>
      i.renderCtx.erase(eraseRegion, xy.ZERO, ...canvases);
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Spectrogram.TYPE]: Spectrogram,
  [FreqAxis.TYPE]: FreqAxis,
  [TimeAxis.TYPE]: TimeAxis,
  [ColorBar.TYPE]: ColorBar,
  [Tooltip.TYPE]: Tooltip,
};
