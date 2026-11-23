// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, type dimensions, map, unique } from "@synnaxlabs/x";

import { dimensionsFromMetrics } from "@/text/aether/dimensions";

export interface AtlasProps {
  font: string;
  textColor: color.Crude;
  characters?: string;
}

const PADDING = 2;
const SCALE_FACTOR = 2;

/**
 * A text atlas that caches glyphs and draws them on a grid. Every glyph takes the cell
 * of a digit, centered in it, so digits line up as tabular figures. A glyph wider than
 * a digit takes a cell of its own width.
 */
export class Atlas {
  private readonly atlas: OffscreenCanvas;
  private readonly charDims: dimensions.Dimensions;
  private readonly charMap: Map<string, number>;
  private readonly advances: Map<string, number>;
  private readonly cols: number;
  /** Width of one atlas cell, wide enough to hold any glyph in the set. */
  private readonly cellWidth: number;
  /** Height of one grid cell, tall enough to hold any glyph in the set. */
  private readonly cellHeight: number;
  /** Distance from the top of a cell to the baseline the glyph is drawn on. */
  private readonly baselineOffset: number;
  /** Drop from the origin each text baseline sets to the alphabetic baseline. */
  private readonly baselineShifts: Record<CanvasTextBaseline, number>;
  private static readonly DEFAULT_CHARS =
    "0123456789.:+-°μmsNa∞ᴇABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz%";

  constructor(props: AtlasProps) {
    const { font, characters = Atlas.DEFAULT_CHARS, textColor } = props;
    this.charMap = new Map();
    this.advances = new Map();

    const uniqueChars = unique.unique(Array.from(characters));

    const tempCanvas = new OffscreenCanvas(1, 1);
    const ctx = tempCanvas.getContext("2d") as OffscreenCanvasRenderingContext2D;
    ctx.font = font;
    ctx.textBaseline = "alphabetic";
    const metrics = ctx.measureText("0");
    this.charDims = {
      width: Math.ceil(metrics.actualBoundingBoxLeft + metrics.actualBoundingBoxRight),
      height: dimensionsFromMetrics(metrics).height,
    };
    // Width pads out to the cell of a digit, which fillText advances by and
    // measureText must report. Height stays the ink, which callers center on.
    this.charDims.width += PADDING;

    // A digit carries neither the tallest ascender nor any descender, so a cell sized
    // from one clips glyphs like "g" and lets them paint into the cell below, which
    // then copies out with its neighbor. Size the cell from the whole set instead.
    const run = ctx.measureText(characters);
    this.baselineOffset = Math.ceil(Math.abs(run.actualBoundingBoxAscent)) + PADDING;
    this.cellHeight =
      this.baselineOffset + Math.ceil(Math.abs(run.actualBoundingBoxDescent)) + PADDING;

    // Canvas measures ink from the origin the current baseline sets, so the drop to
    // the alphabetic baseline is the difference of the two ascents. Measuring beats
    // deriving it: engines disagree on where "middle" and "top" sit.
    const shift = (baseline: CanvasTextBaseline): number => {
      ctx.textBaseline = baseline;
      const { actualBoundingBoxAscent: ascent } = ctx.measureText("0");
      return metrics.actualBoundingBoxAscent - ascent;
    };
    this.baselineShifts = {
      alphabetic: shift("alphabetic"),
      bottom: shift("bottom"),
      hanging: shift("hanging"),
      ideographic: shift("ideographic"),
      middle: shift("middle"),
      top: shift("top"),
    };
    ctx.textBaseline = "alphabetic";

    // Each glyph is drawn so that its ink sits in the middle of its cell.
    const inkLefts = new Map<string, number>();
    uniqueChars.forEach((char) => {
      const m = ctx.measureText(char);
      const ink = Math.ceil(m.actualBoundingBoxLeft + m.actualBoundingBoxRight);
      const advance = Math.max(this.charDims.width, ink + PADDING);
      this.advances.set(char, advance);
      inkLefts.set(char, (advance - ink) / 2 + m.actualBoundingBoxLeft);
    });
    this.cellWidth = Math.max(...this.advances.values());

    const totalChars = uniqueChars.length;
    const cols = Math.ceil(Math.sqrt(totalChars));
    const rows = Math.ceil(totalChars / cols);
    this.cols = cols;

    this.atlas = new OffscreenCanvas(
      this.cellWidth * cols * SCALE_FACTOR,
      this.cellHeight * rows * SCALE_FACTOR,
    );

    const atlasCtx = this.atlas.getContext("2d") as OffscreenCanvasRenderingContext2D;
    atlasCtx.scale(SCALE_FACTOR, SCALE_FACTOR);
    atlasCtx.font = font;
    atlasCtx.textBaseline = "alphabetic";
    atlasCtx.textAlign = "left";
    atlasCtx.fillStyle = color.hex(textColor);
    atlasCtx.clearRect(0, 0, this.atlas.width, this.atlas.height);

    uniqueChars.forEach((char, i) => {
      const col = i % cols;
      const row = Math.floor(i / cols);
      const x = col * this.cellWidth + (inkLefts.get(char) ?? 0);
      const y = row * this.cellHeight + this.baselineOffset;
      atlasCtx.fillText(char, x, y);
      this.charMap.set(char, i);
    });
  }

  private advance(char: string): number {
    return this.advances.get(char) ?? this.charDims.width;
  }

  fillText(
    ctx: OffscreenCanvasRenderingContext2D,
    text: string,
    x: number,
    y: number,
  ): void {
    const { cols, cellWidth, cellHeight } = this;
    const totalWidth = this.measureText(text).width;
    if (ctx.textAlign === "center") x -= totalWidth / 2;
    else if (ctx.textAlign === "right" || ctx.textAlign === "end") x -= totalWidth;
    const top = y + this.baselineShifts[ctx.textBaseline] - this.baselineOffset;

    for (const char of text) {
      const width = this.advance(char);
      const index = this.charMap.get(char);
      if (index !== undefined) {
        const col = index % cols;
        const row = Math.floor(index / cols);
        ctx.drawImage(
          this.atlas,
          col * cellWidth * SCALE_FACTOR,
          row * cellHeight * SCALE_FACTOR,
          width * SCALE_FACTOR,
          cellHeight * SCALE_FACTOR,
          x,
          top,
          width,
          cellHeight,
        );
      }
      x += width;
    }
  }

  measureText(text: string): dimensions.Dimensions {
    let width = 0;
    for (const char of text) width += this.advance(char);
    return { width, height: this.charDims.height };
  }
}

/** A registry for caching atlases for use across multiple components. */
export class AtlasRegistry {
  private readonly atlases: Map<string, Atlas>;

  constructor() {
    this.atlases = new Map();
  }

  /**
   * @returns at atlas from the registry compatible with the given props. If the
   * atlas does not exist in the registry, it is created and added to the registry.
   */
  get(props: AtlasProps): Atlas {
    const key = `${props.font}-${color.hex(props.textColor)}-${props.characters}`;
    return map.getOrInsert(this.atlases, key, () => new Atlas(props));
  }

  /** Drops every atlas, so that the next {@link get} draws its glyphs again. */
  clear(): void {
    this.atlases.clear();
  }
}
