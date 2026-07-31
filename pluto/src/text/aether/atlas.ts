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
// Pixels the baseline sits above the requested y. Preserved from the original
// layout; axes, gauges and values are all positioned against it.
const BASELINE_OFFSET = 2;

/**
 * @desc a text atlas that allows for efficient caching and rendering of monospaced
 * characters.
 */
export class MonospacedAtlas {
  // A canvas buffer that holds rendered characters.
  private readonly atlas: OffscreenCanvas;
  // Cached dimensions of a character.
  private readonly charDims: dimensions.Dimensions;
  // A map of characters to their index in the atlas.
  private readonly charMap: Map<string, number>;
  // Distance from the top of a cell to the baseline of the glyph drawn in it.
  private readonly ascent: number;
  // Vertical stride between cells, sized for descenders unlike charDims.height.
  private readonly cellHeight: number;
  // Source rect of each glyph in the atlas, by index, pre-scaled. Resolving these up
  // front keeps the per-glyph draw to two array reads.
  private readonly srcX: Int32Array;
  private readonly srcY: Int32Array;
  private readonly srcWidth: number;
  private readonly srcHeight: number;
  // The default characters to include in the atlas.
  private static readonly DEFAULT_CHARS =
    "0123456789.:-°µmsNa∞ᴇABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz%";

  constructor(props: AtlasProps) {
    const { font, characters = MonospacedAtlas.DEFAULT_CHARS, textColor } = props;
    this.charMap = new Map();

    const uniqueChars = unique.unique(Array.from(characters));

    const tempCanvas = new OffscreenCanvas(1, 1);
    const ctx = tempCanvas.getContext("2d") as OffscreenCanvasRenderingContext2D;
    ctx.font = font;
    const metrics = ctx.measureText("0");
    this.charDims = dimensionsFromMetrics(metrics);
    this.charDims.width += PADDING;
    this.charDims.height += PADDING;

    // Measured across the whole set, not "0": a digit has no descender, so sizing
    // cells by one clips tails like g and y and bleeds them into the cell below.
    const setMetrics = ctx.measureText(uniqueChars.join(""));
    this.ascent = Math.ceil(Math.abs(setMetrics.actualBoundingBoxAscent)) + PADDING;
    const descent = Math.ceil(Math.abs(setMetrics.actualBoundingBoxDescent)) + PADDING;
    this.cellHeight = this.ascent + descent;

    const totalChars = uniqueChars.length;
    const atlasCharWidth = this.charDims.width;
    this.srcX = new Int32Array(totalChars);
    this.srcY = new Int32Array(totalChars);
    this.srcWidth = atlasCharWidth * SCALE_FACTOR;
    this.srcHeight = this.cellHeight * SCALE_FACTOR;

    const cols = Math.ceil(Math.sqrt(totalChars));
    const rows = Math.ceil(totalChars / cols);

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
      const y = row * this.cellHeight + this.ascent;
      atlasCtx.fillText(char, x, y);
      this.charMap.set(char, i);
      this.srcX[i] = x * SCALE_FACTOR;
      this.srcY[i] = row * this.cellHeight * SCALE_FACTOR;
    });
  }

  fillText(
    ctx: OffscreenCanvasRenderingContext2D,
    text: string,
    x: number,
    y: number,
  ): void {
    const { width, height } = this.charDims;
    if (ctx.textAlign === "center") x -= (width * text.length) / 2;
    if (ctx.textBaseline === "middle") y += height / 2;
    const top = y - this.ascent - BASELINE_OFFSET;

    for (let i = 0; i < text.length; i++) {
      const index = this.charMap.get(text[i]);
      if (index === undefined) continue;

      ctx.drawImage(
        this.atlas,
        this.srcX[index],
        this.srcY[index],
        this.srcWidth,
        this.srcHeight,
        x + i * width,
        top,
        width,
        this.cellHeight,
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
