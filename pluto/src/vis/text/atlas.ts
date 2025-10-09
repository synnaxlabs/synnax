// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, type dimensions, unique } from "@synnaxlabs/x";

import { dimensionsFromMetrics } from "@/text/core/dimensions";

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
  // A canvas buffer that holds rendered characters.
  private readonly atlas: OffscreenCanvas;
  // Cached dimensions of a character.
  private readonly charDims: dimensions.Dimensions;
  // A map of characters to their index in the atlas.
  private readonly charMap: Map<string, number>;
  // The default characters to include in the atlas.
  private static readonly DEFAULT_CHARS =
    "0123456789.:-µmsNa∞ᴇABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz%";

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

    const totalChars = uniqueChars.length;
    const atlasCharWidth = this.charDims.width;
    const atlasCharHeight = this.charDims.height;

    const cols = Math.ceil(Math.sqrt(totalChars));
    const rows = Math.ceil(totalChars / cols);

    this.atlas = new OffscreenCanvas(
      atlasCharWidth * cols * SCALE_FACTOR,
      atlasCharHeight * (rows + 1) * SCALE_FACTOR,
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
      const y = (row + 1) * atlasCharHeight;
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
    const { width, height } = this.charDims;
    const cols = Math.ceil(Math.sqrt(this.charMap.size));
    if (ctx.textAlign === "center") x -= (width * text.length) / 2;
    if (ctx.textBaseline === "middle") y += height / 2;

    for (let i = 0; i < text.length; i++) {
      const char = text[i];
      const index = this.charMap.get(char);
      if (index === undefined) continue;

      const col = index % cols;
      const row = Math.floor(index / cols);
      ctx.drawImage(
        this.atlas,
        col * width * SCALE_FACTOR,
        row * height * SCALE_FACTOR + PADDING,
        width * SCALE_FACTOR,
        height * SCALE_FACTOR,
        x + i * width,
        y - height - PADDING / SCALE_FACTOR,
        width,
        height,
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
