// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, color, location, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { telem } from "@/telem/aether";
import { theming } from "@/theming/aether";
import { Draw2D } from "@/vis/draw2d";
import { type FillTextOptions } from "@/vis/draw2d/canvas";
import { render } from "@/vis/render";
import { type colorMapZ, getLUT } from "@/vis/spectrogram/aether/colormap";
import { FFTProcessor, type windowFunctionZ } from "@/vis/spectrogram/aether/fft";

const FILL_TEXT_OPTIONS: FillTextOptions = { useAtlas: true };
const TICK_LINE_SIZE = 5;
const TICK_PADDING = 6;
const COLOR_BAR_WIDTH = 12;
const COLOR_BAR_GAP = 8;
const TOP_PADDING = 20;
const TOOLTIP_LIST_OFFSET: xy.XY = xy.construct(12);
const TOOLTIP_LIST_SPACING = 3;
const TOOLTIP_LIST_ITEM_HEIGHT = 14;
const TOOLTIP_PADDING: xy.XY = xy.construct(6);

const spectrogramState = z.object({
  box: box.box,
  telem: telem.seriesSourceSpecZ.default(telem.noopSeriesSourceSpec),
  sampleRate: z.number().default(48000),
  fftSize: z.number().default(2048),
  windowFunction: z
    .enum(["hann", "blackmanHarris"] satisfies [
      z.infer<typeof windowFunctionZ>,
      ...Array<z.infer<typeof windowFunctionZ>>,
    ])
    .default("hann"),
  overlap: z.number().min(0).max(0.95).default(0.5),
  colorMap: z
    .enum(["viridis", "inferno", "magma", "plasma", "jet", "grayscale"] satisfies [
      z.infer<typeof colorMapZ>,
      ...Array<z.infer<typeof colorMapZ>>,
    ])
    .default("viridis"),
  dbMin: z.number().default(-100),
  dbMax: z.number().default(0),
  freqMin: z.number().default(0),
  freqMax: z.number().default(0),
  visible: z.boolean().default(true),
  cursorPosition: xy.xy.nullable().default(null),
});

const CANVAS_VARIANTS: render.Canvas2DVariant[] = ["upper2d", "lower2d"];

interface InternalState {
  renderCtx: render.Context;
  theme: theming.Theme;
  draw: Draw2D;
  telem: telem.SeriesSource;
  stopListening?: () => void;
  fft: FFTProcessor;
  sampleBuffer: Float32Array;
  orderedBuffer: Float32Array;
  bufferWritePos: number;
  spectrogramCanvas: OffscreenCanvas;
  spectrogramCtx: OffscreenCanvasRenderingContext2D;
  columnImageData: ImageData;
  hopSize: number;
  samplesUntilNextFFT: number;
  spectrogramWidth: number;
  processedSamples: number;
  leftAxisSize: number;
  bottomAxisSize: number;
  rightAxisSize: number;
}

export class Spectrogram extends aether.Leaf<typeof spectrogramState, InternalState> {
  static readonly TYPE = "spectrogram";
  static readonly z = spectrogramState;
  schema = Spectrogram.z;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.renderCtx = render.Context.use(ctx);
    i.theme = theming.use(ctx);
    i.draw = new Draw2D(i.renderCtx.upper2d, i.theme);

    const { fftSize, windowFunction, overlap } = this.state;
    const hopSize = Math.max(1, Math.round(fftSize * (1 - overlap)));

    if (i.fft == null || i.hopSize !== hopSize || i.sampleBuffer?.length !== fftSize) {
      i.fft = new FFTProcessor(fftSize, windowFunction);
      i.sampleBuffer = new Float32Array(fftSize);
      i.orderedBuffer = new Float32Array(fftSize);
      i.bufferWritePos = 0;
      i.hopSize = hopSize;
      i.samplesUntilNextFFT = fftSize;
      i.processedSamples = 0;
    }

    if (i.leftAxisSize == null) i.leftAxisSize = 50;
    if (i.bottomAxisSize == null) i.bottomAxisSize = 24;
    if (i.rightAxisSize == null) i.rightAxisSize = 60;

    const plotRegion = this.plotRegion();
    const width = Math.max(1, Math.round(box.width(plotRegion)));
    const halfFFT = fftSize / 2;
    if (
      i.spectrogramCanvas == null ||
      i.spectrogramWidth !== width ||
      i.columnImageData?.height !== halfFFT
    ) {
      const oldCanvas = i.spectrogramCanvas;
      const oldWidth = i.spectrogramWidth;
      i.spectrogramWidth = width;
      i.spectrogramCanvas = new OffscreenCanvas(width, halfFFT);
      const sctx = i.spectrogramCanvas.getContext("2d");
      if (sctx == null) throw new Error("Could not get spectrogram 2d context");
      i.spectrogramCtx = sctx;
      if (oldCanvas != null) {
        const copyW = Math.min(width, oldWidth);
        sctx.drawImage(
          oldCanvas,
          oldWidth - copyW,
          0,
          copyW,
          oldCanvas.height,
          width - copyW,
          0,
          copyW,
          halfFFT,
        );
      }
      i.columnImageData = new ImageData(1, halfFFT);
    }

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
    const left = leftAxisSize ?? 50;
    const right = rightAxisSize ?? 60;
    return box.construct(
      { x: box.left(b) + left, y: box.top(b) + TOP_PADDING },
      {
        width: Math.max(1, box.width(b) - left - right),
        height: Math.max(1, box.height(b) - TOP_PADDING - (bottomAxisSize ?? 24)),
      },
    );
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
    const { fftSize, colorMap, dbMin, dbMax, sampleRate, freqMin } = this.state;
    const freqMax = this.state.freqMax > 0 ? this.state.freqMax : sampleRate / 2;
    const halfFFT = fftSize / 2;
    const binFreqStep = sampleRate / fftSize;
    const minBin = Math.max(0, Math.floor(freqMin / binFreqStep));
    const maxBin = Math.min(halfFFT, Math.ceil(freqMax / binFreqStep));
    const invDbRange = 1 / (dbMax - dbMin);
    const lut = getLUT(colorMap);

    let totalSamples = 0;
    for (const series of multiSeries.series) {
      const d = series.data;
      if (d instanceof Float32Array || d instanceof Float64Array)
        totalSamples += d.length;
    }

    let skip = Math.max(0, i.processedSamples);

    for (const series of multiSeries.series) {
      const data = series.data;
      if (!(data instanceof Float32Array) && !(data instanceof Float64Array)) continue;
      const samples = data instanceof Float64Array ? new Float32Array(data) : data;

      let pos = 0;
      if (skip >= samples.length) {
        skip -= samples.length;
        continue;
      }
      if (skip > 0) {
        pos = skip;
        skip = 0;
      }

      while (pos < samples.length) {
        const canWrite = Math.min(
          samples.length - pos,
          i.samplesUntilNextFFT,
          fftSize - i.bufferWritePos,
        );
        i.sampleBuffer.set(samples.subarray(pos, pos + canWrite), i.bufferWritePos);
        i.bufferWritePos = (i.bufferWritePos + canWrite) % fftSize;
        i.samplesUntilNextFFT -= canWrite;
        pos += canWrite;

        if (i.samplesUntilNextFFT <= 0) {
          const ordered = i.orderedBuffer;
          const start = i.bufferWritePos;
          ordered.set(i.sampleBuffer.subarray(start));
          ordered.set(i.sampleBuffer.subarray(0, start), fftSize - start);

          const magnitudes = i.fft.process(ordered);

          const imgData = i.columnImageData.data;
          for (let bin = minBin; bin < maxBin; bin++) {
            const normalized = Math.max(
              0,
              Math.min(1, (magnitudes[bin] - dbMin) * invDbRange),
            );
            const lutIdx =
              Math.max(0, Math.min(255, (normalized * 255 + 0.5) | 0)) << 2;
            const offset = (halfFFT - 1 - bin) << 2;
            imgData[offset] = lut[lutIdx];
            imgData[offset + 1] = lut[lutIdx + 1];
            imgData[offset + 2] = lut[lutIdx + 2];
            imgData[offset + 3] = 255;
          }

          const sctx = i.spectrogramCtx;
          const w = i.spectrogramWidth;
          sctx.drawImage(i.spectrogramCanvas, -1, 0);
          sctx.putImageData(i.columnImageData, w - 1, 0);

          i.samplesUntilNextFFT = i.hopSize;
        }
      }
    }

    i.processedSamples = totalSamples;

    const plotRegion = this.plotRegion();

    const clearCanvasScissor = i.renderCtx.scissor(
      box.construct(b),
      xy.ZERO,
      CANVAS_VARIANTS,
    );

    const clearPlotScissor = i.renderCtx.scissor(plotRegion, xy.ZERO, ["lower2d"]);
    i.renderCtx.lower2d.drawImage(
      i.spectrogramCanvas,
      0,
      halfFFT - maxBin,
      i.spectrogramWidth,
      maxBin - minBin,
      box.left(plotRegion),
      box.top(plotRegion),
      box.width(plotRegion),
      box.height(plotRegion),
    );
    clearPlotScissor();

    this.renderLeftAxis(freqMin, freqMax, plotRegion);
    this.renderBottomAxis(plotRegion);
    this.renderColorBar(plotRegion);
    this.renderCursor(plotRegion, freqMin, freqMax, minBin, maxBin);

    clearCanvasScissor();

    const eraseRegion = box.copy(b);
    return ({ canvases }) =>
      i.renderCtx.erase(eraseRegion, xy.ZERO, ...canvases);
  }

  private renderLeftAxis(freqMin: number, freqMax: number, plot: box.Box): void {
    const { internal: i } = this;
    const canvas = i.renderCtx.lower2d;
    const { theme } = i;

    const axisColor = color.hex(theme.colors.gray.l10);
    const font = theming.fontString(theme, { level: "small", code: true });
    canvas.font = font;
    canvas.strokeStyle = axisColor;
    canvas.fillStyle = axisColor;

    const plotLeft = box.left(plot);
    const plotTop = box.top(plot);
    const plotHeight = box.height(plot);

    // Axis line along the left edge of the plot.
    canvas.beginPath();
    canvas.moveTo(plotLeft, plotTop);
    canvas.lineTo(plotLeft, plotTop + plotHeight);
    canvas.stroke();

    // Tick marks and labels.
    const numTicks = Math.max(2, Math.floor(plotHeight / 75));
    let maxLabelWidth = 0;

    canvas.beginPath();
    for (let j = 0; j <= numTicks; j++) {
      const frac = j / numTicks;
      const freq = freqMin + (freqMax - freqMin) * (1 - frac);
      const y = plotTop + plotHeight * frac;
      const label =
        freq >= 1000 ? `${(freq / 1000).toFixed(1)}k` : `${Math.round(freq)}`;
      const d = canvas.textDimensions(label, FILL_TEXT_OPTIONS);
      if (d.width > maxLabelWidth) maxLabelWidth = d.width;

      // Tick mark pointing left from the axis line.
      canvas.moveTo(plotLeft, y);
      canvas.lineTo(plotLeft - TICK_LINE_SIZE, y);

      // Label to the left of the tick.
      canvas.fillText(
        label,
        plotLeft - d.width - TICK_LINE_SIZE * 2,
        y + d.height / 2,
        undefined,
        FILL_TEXT_OPTIONS,
      );
    }
    canvas.stroke();

    i.leftAxisSize = maxLabelWidth + TICK_LINE_SIZE * 2 + TICK_PADDING;
  }

  private renderBottomAxis(plot: box.Box): void {
    const { internal: i } = this;
    const canvas = i.renderCtx.lower2d;
    const { theme } = i;

    const axisColor = color.hex(theme.colors.gray.l10);
    const font = theming.fontString(theme, { level: "small", code: true });
    canvas.font = font;
    canvas.strokeStyle = axisColor;
    canvas.fillStyle = axisColor;

    const { sampleRate } = this.state;
    const plotLeft = box.left(plot);
    const plotTop = box.top(plot);
    const plotWidth = box.width(plot);
    const plotHeight = box.height(plot);
    const plotBottom = plotTop + plotHeight;

    // Axis line along the bottom of the plot.
    canvas.beginPath();
    canvas.moveTo(plotLeft, plotBottom);
    canvas.lineTo(plotLeft + plotWidth, plotBottom);
    canvas.stroke();

    // Calculate time span shown.
    const totalSeconds = (i.spectrogramWidth * i.hopSize) / sampleRate;
    const numTicks = Math.max(2, Math.floor(plotWidth / 75));
    let maxTickHeight = 0;

    canvas.beginPath();
    for (let j = 0; j <= numTicks; j++) {
      const frac = j / numTicks;
      const x = plotLeft + plotWidth * frac;
      const seconds = totalSeconds * (frac - 1);
      const label = `${seconds.toFixed(1)}s`;
      const d = canvas.textDimensions(label, FILL_TEXT_OPTIONS);
      if (d.height > maxTickHeight) maxTickHeight = d.height;

      // Tick mark pointing down from the axis line.
      canvas.moveTo(x, plotBottom);
      canvas.lineTo(x, plotBottom + TICK_LINE_SIZE);

      // Label centered below the tick.
      canvas.fillText(
        label,
        x - d.width / 2,
        plotBottom + TICK_LINE_SIZE + d.height + TICK_PADDING,
        undefined,
        FILL_TEXT_OPTIONS,
      );
    }
    canvas.stroke();

    i.bottomAxisSize = maxTickHeight + TICK_LINE_SIZE + TICK_PADDING;
  }

  private renderColorBar(plot: box.Box): void {
    const { internal: i } = this;
    const canvas = i.renderCtx.lower2d;
    const { theme } = i;
    const { colorMap, dbMin, dbMax } = this.state;

    const plotRight = box.left(plot) + box.width(plot);
    const plotTop = box.top(plot);
    const plotHeight = box.height(plot);
    const barLeft = plotRight + COLOR_BAR_GAP;
    const lut = getLUT(colorMap);
    const grad = canvas.createLinearGradient(0, plotTop, 0, plotTop + plotHeight);
    const GRADIENT_STOPS = 32;
    for (let j = 0; j <= GRADIENT_STOPS; j++) {
      const frac = j / GRADIENT_STOPS;
      const lutIdx = Math.max(0, Math.min(255, Math.round((1 - frac) * 255))) * 4;
      grad.addColorStop(frac, `rgb(${lut[lutIdx]},${lut[lutIdx + 1]},${lut[lutIdx + 2]})`);
    }
    canvas.fillStyle = grad;
    canvas.fillRect(barLeft, plotTop, COLOR_BAR_WIDTH, plotHeight);

    const axisColor = color.hex(theme.colors.gray.l10);
    const font = theming.fontString(theme, { level: "small", code: true });
    canvas.font = font;
    canvas.strokeStyle = axisColor;
    canvas.fillStyle = axisColor;

    canvas.strokeRect(barLeft, plotTop, COLOR_BAR_WIDTH, plotHeight);

    const tickLeft = barLeft + COLOR_BAR_WIDTH;
    const numTicks = Math.max(2, Math.floor(plotHeight / 75));
    let maxLabelWidth = 0;

    canvas.beginPath();
    for (let j = 0; j <= numTicks; j++) {
      const frac = j / numTicks;
      const db = dbMax + (dbMin - dbMax) * frac;
      const y = plotTop + plotHeight * frac;
      const label = `${Math.round(db)}`;
      const d = canvas.textDimensions(label, FILL_TEXT_OPTIONS);
      if (d.width > maxLabelWidth) maxLabelWidth = d.width;

      canvas.moveTo(tickLeft, y);
      canvas.lineTo(tickLeft + TICK_LINE_SIZE, y);

      canvas.fillText(
        label,
        tickLeft + TICK_LINE_SIZE * 2,
        y + d.height / 2,
        undefined,
        FILL_TEXT_OPTIONS,
      );
    }
    canvas.stroke();

    const dbLabel = "dB";
    const dbD = canvas.textDimensions(dbLabel, FILL_TEXT_OPTIONS);
    canvas.fillText(
      dbLabel,
      barLeft + (COLOR_BAR_WIDTH - dbD.width) / 2,
      plotTop - TICK_PADDING,
      undefined,
      FILL_TEXT_OPTIONS,
    );

    i.rightAxisSize =
      COLOR_BAR_GAP +
      COLOR_BAR_WIDTH +
      TICK_LINE_SIZE * 2 +
      maxLabelWidth +
      TICK_PADDING;
  }

  private renderCursor(
    plot: box.Box,
    freqMin: number,
    freqMax: number,
    minBin: number,
    maxBin: number,
  ): void {
    const { cursorPosition } = this.state;
    if (cursorPosition == null) return;
    const { internal: i } = this;

    const plotLeft = box.left(plot);
    const plotTop = box.top(plot);
    const plotWidth = box.width(plot);
    const plotHeight = box.height(plot);
    const cx = cursorPosition.x;
    const cy = cursorPosition.y;

    if (
      cx < plotLeft ||
      cx > plotLeft + plotWidth ||
      cy < plotTop ||
      cy > plotTop + plotHeight
    )
      return;

    const { draw } = i;
    const ruleColor = i.theme.colors.gray.l7;

    draw.rule({
      stroke: ruleColor,
      lineWidth: 1,
      lineDash: 0,
      direction: "y",
      region: plot,
      position: cx,
    });

    draw.rule({
      stroke: ruleColor,
      lineWidth: 1,
      lineDash: 0,
      direction: "x",
      region: plot,
      position: cy,
    });

    const yFrac = (cy - plotTop) / plotHeight;
    const freq = freqMin + (1 - yFrac) * (freqMax - freqMin);

    const { fftSize, sampleRate, colorMap, dbMin, dbMax } = this.state;
    const halfFFT = fftSize / 2;
    const binFreqStep = sampleRate / fftSize;
    const bin = Math.round(freq / binFreqStep);
    const canvasX = Math.round(
      ((cx - plotLeft) / plotWidth) * (i.spectrogramWidth - 1),
    );
    const pixelY = halfFFT - 1 - Math.max(minBin, Math.min(maxBin - 1, bin));

    let db = NaN;
    if (
      canvasX >= 0 &&
      canvasX < i.spectrogramWidth &&
      pixelY >= 0 &&
      pixelY < halfFFT
    ) {
      const pixel = i.spectrogramCtx.getImageData(canvasX, pixelY, 1, 1).data;
      const pr = pixel[0];
      const pg = pixel[1];
      const pb = pixel[2];
      const lut = getLUT(colorMap);
      let bestDist = Infinity;
      let bestIdx = 0;
      for (let j = 0; j < 256; j++) {
        const li = j << 2;
        const dr = pr - lut[li];
        const dg = pg - lut[li + 1];
        const dbl = pb - lut[li + 2];
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

    const xFrac = (cx - plotLeft) / plotWidth;
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

export const REGISTRY: aether.ComponentRegistry = {
  [Spectrogram.TYPE]: Spectrogram,
};
