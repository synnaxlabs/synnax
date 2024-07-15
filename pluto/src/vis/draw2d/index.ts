// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  box,
  Destructor,
  type dimensions,
  direction,
  location,
  toArray,
  xy,
} from "@synnaxlabs/x";

import { color } from "@/color/core";
import { type text } from "@/text/core";
import { dimensions as textDimensions } from "@/text/dimensions";
import { type theming } from "@/theming/aether";
import { fontString } from "@/theming/core/fontString";
import { SugaredOffscreenCanvasRenderingContext2D } from "@/vis/draw2d/canvas";

export interface Draw2DLineProps {
  stroke: color.Color;
  lineWidth: number;
  lineDash: number;
  start: xy.XY;
  end: xy.XY;
}

export interface Draw2DRuleProps extends Omit<Draw2DLineProps, "start" | "end"> {
  direction: direction.Direction;
  region: box.Box;
  position: number;
}

export interface Draw2DCircleProps {
  fill: color.Color;
  radius: number;
  position: xy.XY;
}

export interface Draw2DContainerProps {
  region: box.Box;
  bordered?: boolean | location.Location | location.Location[];
  rounded?: boolean;
  borderColor?: ColorSpec;
  borderRadius?: number;
  borderWidth?: number;
  backgroundColor?: ColorSpec;
}

export interface DrawTextProps {
  text: string;
  position: xy.XY;
  level: text.Level;
  weight?: text.Weight;
  shade?: text.Shade;
  maxWidth?: number;
}

export interface DrawTextInCenterProps
  extends Omit<DrawTextProps, "position" | "direction"> {
  box: box.Box;
}

export interface Draw2DMeasureTextContainerProps {
  text: string[];
  direction: direction.Direction;
  level: text.Level;
  spacing?: number;
}

export interface Draw2DBorderProps {
  region: box.Box;
  color?: ColorSpec;
  width?: number;
  radius?: number;
  location?: true | location.Location | location.Location[];
}

export interface Draw2DTextContainerProps
  extends Omit<Draw2DContainerProps, "region">,
    Draw2DMeasureTextContainerProps {
  position: xy.XY;
  offset?: xy.XY;
  root?: location.CornerXY;
}

type ColorSpec = color.Crude | ((t: theming.Theme) => color.Color);

export class Draw2D {
  readonly canvas: SugaredOffscreenCanvasRenderingContext2D;
  readonly theme: theming.Theme;

  constructor(canvas: SugaredOffscreenCanvasRenderingContext2D, theme: theming.Theme) {
    this.canvas = canvas;
    this.theme = theme;
  }

  rule({ direction, region, position, ...props }: Draw2DRuleProps): void {
    if (direction === "x")
      return this.line({
        start: xy.construct(box.left(region), position),
        end: xy.construct(box.right(region), position),
        ...props,
      });
    return this.line({
      start: xy.construct(position, box.top(region)),
      end: xy.construct(position, box.bottom(region)),
      ...props,
    });
  }

  line({ stroke, lineWidth, lineDash, start, end }: Draw2DLineProps): void {
    const ctx = this.canvas;
    ctx.strokeStyle = stroke.hex;
    ctx.lineWidth = lineWidth;
    ctx.setLineDash([lineDash]);
    ctx.beginPath();
    ctx.moveTo(...xy.couple(start));
    ctx.lineTo(...xy.couple(end));
    ctx.stroke();
  }

  circle({ fill, radius, position }: Draw2DCircleProps): void {
    const ctx = this.canvas;
    ctx.fillStyle = fill.hex;
    ctx.beginPath();
    ctx.arc(...xy.couple(position), radius, 0, 2 * Math.PI);
    ctx.fill();
  }

  resolveColor(c: ColorSpec | undefined, fallback: ColorSpec): color.Color;

  resolveColor(c: ColorSpec): color.Color;

  resolveColor(c: ColorSpec | undefined, fallback?: ColorSpec): color.Color {
    if (c == null) return this.resolveColor(fallback as ColorSpec);
    if (typeof c === "function") return c(this.theme);
    return new color.Color(c);
  }

  border({ region, color, width, radius, location }: Draw2DBorderProps): void {
    const ctx = this.canvas;
    ctx.strokeStyle = this.resolveColor(color, this.theme.colors.border).hex;
    ctx.lineWidth = width ?? this.theme.sizes.border.width;
    radius ??= this.theme.sizes.border.radius;
    if (location == null || location === true) {
      if (radius > 0) {
        ctx.roundRect(
          ...xy.couple(box.topLeft(region)),
          ...xy.couple(box.dims(region)),
          radius,
        );

        ctx.stroke();
      } else {
        ctx.rect(...xy.couple(box.topLeft(region)), ...xy.couple(box.dims(region)));
        ctx.stroke();
      }
    } else
      toArray(location).forEach((loc) => {
        const [start, end] = box.edgePoints(region, loc);
        ctx.beginPath();
        ctx.moveTo(...xy.couple(start));
        ctx.lineTo(...xy.couple(end));
        ctx.stroke();
      });
  }

  container({
    region,
    bordered = true,
    rounded = true,
    borderColor,
    borderRadius,
    borderWidth,
    backgroundColor,
  }: Draw2DContainerProps): void {
    if (borderRadius == null) borderRadius = this.theme.sizes.border.radius;
    if (borderWidth == null) borderWidth = 1;
    const ctx = this.canvas;
    ctx.fillStyle = this.resolveColor(backgroundColor, this.theme.colors.gray.l1).hex;
    ctx.strokeStyle = this.resolveColor(borderColor, this.theme.colors.border).hex;
    ctx.setLineDash([]);
    ctx.lineWidth = 1;
    ctx.beginPath();
    if (rounded)
      ctx.roundRect(
        ...xy.couple(box.topLeft(region)),
        ...xy.couple(box.dims(region)),
        borderRadius,
      );
    else ctx.rect(...xy.couple(box.topLeft(region)), ...xy.couple(box.dims(region)));
    ctx.fill();
    if (bordered)
      this.border({
        region,
        color: borderColor,
        radius: borderRadius,
        width: borderWidth,
        location: bordered,
      });
  }

  textContainer(props: Draw2DTextContainerProps): void {
    const [dims, draw] = this.spacedTextDrawF(props);
    dims.width += 12;
    dims.height += 12;
    const { root = location.TOP_LEFT, offset = xy.ZERO } = props;
    const position = { ...props.position };
    if (root.x === "right") {
      position.x -= dims.width + offset.x;
    } else {
      position.x += offset.x;
    }
    if (root.y === "bottom") {
      position.y -= dims.height + offset.y;
    } else {
      position.y += offset.y;
    }
    this.container({
      region: box.construct(position, dims.width, dims.height),
      ...props,
    });
    this.canvas.filter = "none";
    draw(xy.translate(position, [6, 6]));
  }

  spacedTextDrawF({
    text,
    direction: d,
    spacing = 1,
    level = "p",
  }: Draw2DMeasureTextContainerProps): [dimensions.Dimensions, (base: xy.XY) => void] {
    const font = fontString(this.theme, level);
    const textDims = text.map((t) => textDimensions(t, font, this.canvas));
    const spacingPx = this.theme.sizes.base * spacing;
    const offset =
      Math.max(...textDims.map((td) => td[direction.dimension(d)])) + spacingPx;
    return [
      {
        [direction.dimension(direction.swap(d)) as "width"]: Math.max(
          ...textDims.map((td) => td[direction.dimension(direction.swap(d))]),
        ),
        [direction.dimension(d) as "height"]: offset * text.length - spacingPx,
      },

      (position: xy.XY) => {
        const font = fontString(this.theme, level);
        this.canvas.font = font;
        this.canvas.fillStyle = this.theme.colors.text.hex;
        this.canvas.textBaseline = "top";
        text.forEach((v, i) => {
          this.canvas.fillText(v, position.x, position.y + offset * i);
        });
      },
    ];
  }

  drawTextInCenter({ box: b, text, level }: DrawTextInCenterProps): void {
    const dims = textDimensions(text, this.canvas.font, this.canvas);
    const pos = box.positionInCenter(box.construct(xy.ZERO, dims), b);
    return this.text({ text, position: box.topLeft(pos), level });
  }

  text({ text, position, level = "p", weight, shade, maxWidth }: DrawTextProps): void {
    this.canvas.font = fontString(this.theme, level, weight);
    if (shade == null) this.canvas.fillStyle = this.theme.colors.text.hex;
    else this.canvas.fillStyle = this.theme.colors.gray[`l${shade}`].hex;
    this.canvas.textBaseline = "top";
    let removeScissor: Destructor | undefined;
    if (maxWidth != null)
      removeScissor = this.canvas.scissor(box.construct(position, maxWidth, 1000));
    this.canvas.fillText(text, position.x, position.y);
    removeScissor?.();
  }
}
