// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  array,
  box,
  color,
  type destructor,
  type dimensions,
  direction,
  location,
  xy,
} from "@synnaxlabs/x";

import { type text } from "@/text/base";
import { dimensions as textDimensions } from "@/text/base/dimensions";
import { type theming } from "@/theming/aether";
import { fontString } from "@/theming/base/fontString";
import {
  type FillTextOptions,
  type SugaredOffscreenCanvasRenderingContext2D,
} from "@/vis/draw2d/canvas";

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
  fill?: color.Color;
  stroke?: color.Color;
  strokeWidth?: number;
  lineDash?: number;
  radius: number | { inner: number; outer: number };
  position: xy.XY;
  angle?: { lower: number; upper: number };
  lineCap?: CanvasLineCap;
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

export interface DrawTextProps extends FillTextOptions {
  text: string;
  position: xy.XY;
  level: text.Level;
  justify?: CanvasTextAlign;
  align?: CanvasTextBaseline;
  weight?: text.Weight;
  shade?: text.Shade;
  maxWidth?: number;
  code?: boolean;
  color?: ColorSpec;
}

export interface DrawTextInCenterProps extends Omit<
  DrawTextProps,
  "position" | "direction"
> {
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
  extends Omit<Draw2DContainerProps, "region">, Draw2DMeasureTextContainerProps {
  position: xy.XY;
  offset?: xy.XY;
  root?: location.CornerXY;
}

export interface DrawList {
  length: number;
  position: xy.XY;
  itemHeight: number;
  spacing?: number;
  width: number;
  draw: (index: number, box: box.Box) => void;
  root?: location.CornerXY;
  offset?: xy.XY;
  padding?: xy.XY;
}

type ColorSpec = color.Crude | ((t: theming.Theme) => color.Color);

export class Draw2D {
  readonly canvas: SugaredOffscreenCanvasRenderingContext2D;
  readonly theme: theming.Theme;

  constructor(canvas: SugaredOffscreenCanvasRenderingContext2D, theme: theming.Theme) {
    this.canvas = canvas;
    this.theme = theme;
  }

  rule({ direction, region, position, ...rest }: Draw2DRuleProps): void {
    if (direction === "x")
      return this.line({
        start: xy.construct(box.left(region), position),
        end: xy.construct(box.right(region), position),
        ...rest,
      });
    return this.line({
      start: xy.construct(position, box.top(region)),
      end: xy.construct(position, box.bottom(region)),
      ...rest,
    });
  }

  line({ stroke, lineWidth, lineDash, start, end }: Draw2DLineProps): void {
    const ctx = this.canvas;
    ctx.strokeStyle = color.hex(stroke);
    ctx.lineWidth = lineWidth;
    ctx.setLineDash([lineDash]);
    ctx.beginPath();
    ctx.moveTo(...xy.couple(start));
    ctx.lineTo(...xy.couple(end));
    ctx.stroke();
  }

  circle({
    fill,
    stroke,
    strokeWidth,
    lineDash,
    radius,
    position,
    angle,
    lineCap,
  }: Draw2DCircleProps): void {
    const ctx = this.canvas;
    ctx.beginPath();
    const startAngle = angle?.lower ?? 0;
    const endAngle = angle?.upper ?? 2 * Math.PI;

    if (stroke != null && typeof radius === "object") {
      // Stroke mode for rings - draw as a thick arc with rounded caps
      const { inner, outer } = radius;
      const midRadius = (inner + outer) / 2;
      const arcWidth = outer - inner;

      ctx.arc(...xy.couple(position), midRadius, startAngle, endAngle, false);
      ctx.strokeStyle = color.hex(stroke);
      ctx.lineWidth = strokeWidth ?? arcWidth;
      if (lineCap) ctx.lineCap = lineCap;
      if (lineDash != null) ctx.setLineDash([lineDash]);
      ctx.stroke();
      if (lineDash != null) ctx.setLineDash([]);
    } else if (stroke != null && typeof radius === "number") {
      // Stroke mode for simple circles
      ctx.arc(...xy.couple(position), radius, startAngle, endAngle, false);
      ctx.strokeStyle = color.hex(stroke);
      ctx.lineWidth = strokeWidth ?? 1;
      if (lineCap) ctx.lineCap = lineCap;
      if (lineDash != null) ctx.setLineDash([lineDash]);
      ctx.stroke();
      if (lineDash != null) ctx.setLineDash([]);
    } else if (fill != null) {
      // Fill mode (original behavior)
      ctx.fillStyle = color.hex(fill);

      if (typeof radius === "number") {
        // Simple filled circle or arc
        ctx.arc(...xy.couple(position), radius, startAngle, endAngle);
        ctx.fill();
      } else {
        // Ring or arc segment with inner and outer radius
        const { inner, outer } = radius;
        // Draw outer arc
        ctx.arc(...xy.couple(position), outer, startAngle, endAngle, false);
        // Draw line to inner arc start
        const innerStartX = position.x + inner * Math.cos(endAngle);
        const innerStartY = position.y + inner * Math.sin(endAngle);
        ctx.lineTo(innerStartX, innerStartY);
        // Draw inner arc (reverse direction)
        ctx.arc(...xy.couple(position), inner, endAngle, startAngle, true);
        ctx.closePath();
        ctx.fill();
      }
    }
  }

  resolveColor(c: ColorSpec | undefined, fallback: ColorSpec): color.Color;

  resolveColor(c: ColorSpec): color.Color;

  resolveColor(c: ColorSpec | undefined, fallback?: ColorSpec): color.Color {
    if (c == null) {
      if (fallback == null) return this.theme.colors.text;
      return this.resolveColor(fallback);
    }
    if (typeof c === "function") return c(this.theme);
    return color.construct(c);
  }

  border({
    region,
    color: colorVal,
    width,
    radius,
    location,
  }: Draw2DBorderProps): void {
    const ctx = this.canvas;
    ctx.strokeStyle = color.hex(this.resolveColor(colorVal, this.theme.colors.border));
    ctx.lineWidth = width ?? this.theme.sizes.border.width;
    radius ??= this.theme.sizes.border.radius;
    if (location == null || location === true)
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
    else
      array.toArray(location).forEach((loc) => {
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
    borderRadius ??= this.theme.sizes.border.radius;
    borderWidth ??= 1;
    const ctx = this.canvas;
    ctx.fillStyle = color.hex(
      this.resolveColor(backgroundColor, this.theme.colors.gray.l1),
    );
    ctx.strokeStyle = color.hex(
      this.resolveColor(borderColor, this.theme.colors.border),
    );
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
    if (root.x === "right") position.x -= dims.width + offset.x;
    else position.x += offset.x;
    if (root.y === "bottom") position.y -= dims.height + offset.y;
    else position.y += offset.y;
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
    const font = fontString(this.theme, { level });
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
        const font = fontString(this.theme, { level });
        this.canvas.font = font;
        this.canvas.fillStyle = color.hex(this.theme.colors.text);
        this.canvas.textBaseline = "top";
        this.canvas.textAlign = "start";
        text.forEach((v, i) => {
          this.canvas.fillText(v, position.x, position.y + offset * i);
        });
      },
    ];
  }

  list({
    length,
    itemHeight,
    width,
    spacing = 0,
    position,
    draw,
    root = location.TOP_LEFT,
    offset = xy.ZERO,
    padding = xy.ZERO,
  }: DrawList): void {
    const height = length * itemHeight + padding.y * 2 + spacing * (length - 1);
    const wid = width + padding.x * 2;
    const pos = { ...position };
    if (root.x === "right") pos.x -= width + offset.x * 2;
    else pos.x += offset.x;
    if (root.y === "top") pos.y -= height + offset.y * 2;
    else pos.y += offset.y;
    this.container({
      region: box.construct(pos, { width: wid, height }),
      backgroundColor: (t) => t.colors.gray.l1,
    });
    for (let i = 0; i < length; i++) {
      const itemBox = box.construct(
        xy.construct(
          pos.x + padding.x,
          pos.y + i * itemHeight + padding.y + spacing * i,
        ),
        width,
        itemHeight,
      );
      draw(i, itemBox);
    }
  }

  drawTextInCenter({ box: b, text, level }: DrawTextInCenterProps): void {
    const dims = textDimensions(text, this.canvas.font, this.canvas);
    const pos = box.positionInCenter(box.construct(xy.ZERO, dims), b);
    return this.text({ text, position: box.topLeft(pos), level });
  }

  text({
    text,
    position,
    level = "p",
    weight,
    shade,
    maxWidth,
    code,
    justify = "left",
    align = "top",
    useAtlas,
    color: colorVal,
  }: DrawTextProps): void {
    this.canvas.font = fontString(this.theme, { level, weight, code });
    if (colorVal != null)
      this.canvas.fillStyle = color.hex(this.resolveColor(colorVal));
    else if (shade == null) this.canvas.fillStyle = color.hex(this.theme.colors.text);
    else this.canvas.fillStyle = color.hex(this.theme.colors.gray[`l${shade}`]);
    this.canvas.textAlign = justify;
    this.canvas.textBaseline = align;
    let removeScissor: destructor.Destructor | undefined;
    if (maxWidth != null)
      removeScissor = this.canvas.scissor(box.construct(position, maxWidth, 1000));
    this.canvas.fillText(text, position.x, position.y, undefined, { useAtlas });
    removeScissor?.();
  }
}
