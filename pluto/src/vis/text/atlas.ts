import { type dimensions, unique } from "@synnaxlabs/x";

import { color } from "@/color/core";

export interface AtlasProps {
  font: string;
  textColor: color.Crude;
  dpr?: number;
  characters?: string;
}

export class Atlas {
  private readonly atlas: OffscreenCanvas;
  private readonly charWidth: number;
  private readonly charHeight: number;
  private readonly dpr: number;
  private readonly charMap: Map<string, number>;
  private static readonly PADDING = 2;
  private static readonly DEFAULT_CHARS = "0123456789.:-ums";

  constructor(props: AtlasProps) {
    const { font, dpr = 2, characters = Atlas.DEFAULT_CHARS, textColor } = props;
    this.dpr = dpr;
    this.charMap = new Map();

    const uniqueChars = unique.unique(Array.from(characters));

    this.atlas = new OffscreenCanvas(1, 1);
    const ctx = this.atlas.getContext("2d") as OffscreenCanvasRenderingContext2D;
    ctx.font = font;
    const metrics = ctx.measureText("0");
    const ascent = metrics.fontBoundingBoxAscent || metrics.actualBoundingBoxAscent;
    const descent = metrics.fontBoundingBoxDescent || metrics.actualBoundingBoxDescent;

    this.charWidth = metrics.width;
    this.charHeight = ascent + descent;

    const totalChars = uniqueChars.length;
    const atlasCharWidth = this.charWidth + Atlas.PADDING * 2;
    const atlasCharHeight = this.charHeight + Atlas.PADDING * 2;

    const cols = Math.ceil(Math.sqrt(totalChars));
    const rows = Math.ceil(totalChars / cols);

    this.atlas = new OffscreenCanvas(
      atlasCharWidth * cols * this.dpr,
      atlasCharHeight * rows * this.dpr,
    );

    const atlasCtx = this.atlas.getContext("2d") as OffscreenCanvasRenderingContext2D;
    atlasCtx.scale(this.dpr, this.dpr);
    atlasCtx.font = font;
    atlasCtx.textBaseline = "alphabetic";
    atlasCtx.textAlign = "left";
    atlasCtx.fillStyle = new color.Color(textColor).hex;
    atlasCtx.clearRect(0, 0, this.atlas.width, this.atlas.height);

    uniqueChars.forEach((char, i) => {
      const col = i % cols;
      const row = Math.floor(i / cols);
      const x = col * atlasCharWidth + Atlas.PADDING;
      const y = row * atlasCharHeight + ascent + Atlas.PADDING;
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
    const atlasCharWidth = this.charWidth + Atlas.PADDING * 2;
    const atlasCharHeight = this.charHeight + Atlas.PADDING * 2;
    const cols = Math.ceil(Math.sqrt(this.charMap.size));

    for (let i = 0; i < text.length; i++) {
      const char = text[i];
      const index = this.charMap.get(char);
      if (index === undefined) continue;

      const col = index % cols;
      const row = Math.floor(index / cols);

      ctx.drawImage(
        this.atlas,
        col * atlasCharWidth * this.dpr,
        row * atlasCharHeight * this.dpr,
        atlasCharWidth * this.dpr,
        atlasCharHeight * this.dpr,
        x + i * this.charWidth,
        y - this.charHeight,
        this.charWidth + Atlas.PADDING * 2,
        this.charHeight + Atlas.PADDING * 2,
      );
    }
  }

  measureText(text: string): dimensions.Dimensions {
    return {
      width: text.length * this.charWidth,
      height: this.charHeight,
    };
  }
}

export class AtlasRegistry {
  private readonly atlases: Map<string, Atlas>;

  constructor() {
    this.atlases = new Map();
  }

  get(props: AtlasProps): Atlas {
    const c = new color.Color(props.textColor);
    const key = `${props.font}-${c.hex}-${props.dpr}-${props.characters}`;
    if (this.atlases.has(key)) return this.atlases.get(key)!;
    const atlas = new Atlas(props);
    this.atlases.set(key, atlas);
    return atlas;
  }
}
