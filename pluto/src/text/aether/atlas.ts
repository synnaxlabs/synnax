// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, type dimensions, unique } from "@synnaxlabs/x";

import { dimensionsFromMetrics } from "@/text/aether/dimensions";

export interface AtlasProps {
  font: string;
  textColor: color.Crude;
  characters?: string;
}

const PADDING = 2;
const SCALE_FACTOR = 2;
/**
 * @desc a text atlas that allows for efficient caching and rendering of monospaced
 * characters.
 */
export class MonospacedAtlas {
  private readonly atlas: OffscreenCanvas;
  private readonly charDims: dimensions.Dimensions;
  private readonly charMap: Map<string, number>;
  private readonly cols: number;
  /** Height of one grid cell, tall enough to hold any glyph in the set. */
  private readonly cellHeight: number;
  /** Distance from the top of a cell to the baseline the glyph is drawn on. */
  private readonly baselineOffset: number;
  /** Drop from the origin each text baseline sets to the alphabetic baseline. */
  private readonly baselineShifts: Record<CanvasTextBaseline, number>;
  private static readonly DEFAULT_CHARS =
    "0123456789.:-°µmsNa∞ᴇABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz%";

  constructor(props: AtlasProps) {
    const { font, characters = MonospacedAtlas.DEFAULT_CHARS, textColor } = props;
    this.charMap = new Map();

    const uniqueChars = unique.unique(Array.from(characters));

    const tempCanvas = new OffscreenCanvas(1, 1);
    const ctx = tempCanvas.getContext("2d") as OffscreenCanvasRenderingContext2D;
    ctx.font = font;
    ctx.textBaseline = "alphabetic";
    const metrics = ctx.measureText("0");
    this.charDims = dimensionsFromMetrics(metrics);
    // Width pads out to the cell the atlas lays glyphs on, which fillText advances by
    // and measureText must report. Height stays the ink, which callers center on.
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

    const totalChars = uniqueChars.length;
    const atlasCharWidth = this.charDims.width;

    const cols = Math.ceil(Math.sqrt(totalChars));
    const rows = Math.ceil(totalChars / cols);
    this.cols = cols;

    this.atlas = new OffscreenCanvas(
      atlasCharWidth * cols * SCALE_FACTOR,
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
      const x = col * atlasCharWidth;
      const y = row * this.cellHeight + this.baselineOffset;
      atlasCtx.fillText(char, x, y);
      this.charMap.set(char, i);
    });
  }

  fillText(
    ctx: OffscreenCanvasRenderingContext2D,
    text: string,
    x: number,
    y: number,
  ): void {
    const { width } = this.charDims;
    const { cols, cellHeight } = this;
    const totalWidth = width * text.length;
    if (ctx.textAlign === "center") x -= totalWidth / 2;
    else if (ctx.textAlign === "right" || ctx.textAlign === "end") x -= totalWidth;
    const top = y + this.baselineShifts[ctx.textBaseline] - this.baselineOffset;

    for (let i = 0; i < text.length; i++) {
      const char = text[i];
      const index = this.charMap.get(char);
      if (index === undefined) continue;

      const col = index % cols;
      const row = Math.floor(index / cols);
      ctx.drawImage(
        this.atlas,
        col * width * SCALE_FACTOR,
        row * cellHeight * SCALE_FACTOR,
        width * SCALE_FACTOR,
        cellHeight * SCALE_FACTOR,
        x + i * width,
        top,
        width,
        cellHeight,
      );
    }
  }

  measureText(text: string): dimensions.Dimensions {
    return { width: text.length * this.charDims.width, height: this.charDims.height };
  }
}

/** A registry for caching atlases for use across multiple components. */
export class AtlasRegistry {
  private readonly atlases: Map<string, MonospacedAtlas>;

  constructor() {
    this.atlases = new Map();
  }

  /**
   * @returns at atlas from the registry compatible with the given props. If the
   * atlas does not exist in the registry, it is created and added to the registry.
   */
  get(props: AtlasProps): MonospacedAtlas {
    const key = `${props.font}-${color.hex(props.textColor)}-${props.characters}`;
    if (this.atlases.has(key)) return this.atlases.get(key)!;
    const atlas = new MonospacedAtlas(props);
    this.atlases.set(key, atlas);
    return atlas;
  }
}
