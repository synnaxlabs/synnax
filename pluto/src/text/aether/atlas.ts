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
// Room left around a glyph's ink for its anti-aliasing.
const INK_MARGIN = 1;
const SCALE_FACTOR = 2;

interface Glyph {
  /** Index of the glyph's cell in the atlas. */
  index: number;
  /** How far the glyph moves the cursor, as the font sets it. */
  advance: number;
  /**
   * Distance from the left of the glyph's cell to its origin. Negative when the ink
   * starts right of the origin.
   */
  origin: number;
  /** Width of the cell region that holds the glyph's ink. */
  width: number;
}

/**
 * A text atlas that caches glyphs and draws text from them. Each glyph is drawn once,
 * and text moves by each glyph's own advance, so it spaces as the font does. A font
 * with tabular digits keeps every digit one width.
 */
export class Atlas {
  private readonly atlas: OffscreenCanvas;
  private readonly glyphs: Map<string, Glyph>;
  /** The advance of a "0", which a glyph outside the set moves the cursor by. */
  private readonly digitAdvance: number;
  /** Ink height of a "0", which callers center on. */
  private readonly inkHeight: number;
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
    " 0123456789.:+-°μmsNa∞ᴇABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz%";

  constructor(props: AtlasProps) {
    const { font, characters = Atlas.DEFAULT_CHARS, textColor } = props;
    this.glyphs = new Map();

    const uniqueChars = unique.unique(Array.from(characters));

    const tempCanvas = new OffscreenCanvas(1, 1);
    const ctx = tempCanvas.getContext("2d") as OffscreenCanvasRenderingContext2D;
    ctx.font = font;
    ctx.textBaseline = "alphabetic";
    const metrics = ctx.measureText("0");
    this.digitAdvance = metrics.width;
    this.inkHeight = dimensionsFromMetrics(metrics).height;

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

    // A copy reads only the glyph's ink, which can start right of its origin and
    // reach past its advance.
    uniqueChars.forEach((char, index) => {
      const m = ctx.measureText(char);
      const origin = Math.ceil(m.actualBoundingBoxLeft) + INK_MARGIN;
      const width = origin + Math.ceil(m.actualBoundingBoxRight) + INK_MARGIN;
      this.glyphs.set(char, { index, advance: m.width, origin, width });
    });
    this.cellWidth = Math.max(...Array.from(this.glyphs.values(), (g) => g.width));

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

    this.glyphs.forEach(({ index, origin }, char) => {
      const col = index % cols;
      const row = Math.floor(index / cols);
      const x = col * this.cellWidth + origin;
      const y = row * this.cellHeight + this.baselineOffset;
      atlasCtx.fillText(char, x, y);
    });
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
      const glyph = this.glyphs.get(char);
      if (glyph == null) {
        x += this.digitAdvance;
        continue;
      }
      const { index, advance, origin, width } = glyph;
      const col = index % cols;
      const row = Math.floor(index / cols);
      ctx.drawImage(
        this.atlas,
        col * cellWidth * SCALE_FACTOR,
        row * cellHeight * SCALE_FACTOR,
        width * SCALE_FACTOR,
        cellHeight * SCALE_FACTOR,
        x - origin,
        top,
        width,
        cellHeight,
      );
      x += advance;
    }
  }

  /** @returns the width text takes, the sum of its glyphs' advances, and ink height. */
  measureText(text: string): dimensions.Dimensions {
    let width = 0;
    for (const char of text)
      width += this.glyphs.get(char)?.advance ?? this.digitAdvance;
    return { width, height: this.inkHeight };
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
