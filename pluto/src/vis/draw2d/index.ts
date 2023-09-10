// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.


import { type color } from "@/color/core";
import { type Level } from "@/text/types";
import { type theming } from "@/theming/aether";
import { fontString } from "@/theming/core/fontString";
import { box, direction, xy, dimensions} from "@synnaxlabs/x";
import { dimensions as textDimensions } from "@/text/dimensions";

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
  bordered?: boolean;
  rounded?: boolean;
  borderColor?: color.Color;
  borderRadius?: number;
  borderWidth?: number;
  backgroundColor?: color.Color;
}

export interface DrawTextProps {
  text: string;
  position: xy.XY;
  level: Level;
  direction: direction.Direction;
}

export interface Draw2DMeasureTextContainerProps {
  text: string[];
  direction: direction.Direction;
  level: Level;
  spacing?: number;
}

export interface Draw2DTextContainerProps
  extends Omit<Draw2DContainerProps, "region">,
    Draw2DMeasureTextContainerProps {
  position: xy.XY;
}

export class Draw2D {
  readonly canvas: OffscreenCanvasRenderingContext2D;
  readonly theme: theming.Theme;

  constructor(canvas: OffscreenCanvasRenderingContext2D, theme: theming.Theme) {
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

  container({
    region,
    bordered = true,
    rounded = true,
    borderColor,
    borderRadius,
    borderWidth,
    backgroundColor,
  }: Draw2DContainerProps): void {
    if (borderColor == null) borderColor = this.theme.colors.border;
    if (backgroundColor == null) backgroundColor = this.theme.colors.gray.m3;
    if (borderRadius == null) borderRadius = this.theme.sizes.border.radius;
    if (borderWidth == null) borderWidth = 1;

    const ctx = this.canvas;
    ctx.fillStyle = backgroundColor.hex;
    ctx.strokeStyle = borderColor.hex;
    ctx.setLineDash([]);
    ctx.lineWidth = 1;
    ctx.beginPath();
    if (rounded)
      ctx.roundRect(...xy.couple(box.topLeft(region)), ...xy.couple(box.dims(region)), borderRadius);
    else ctx.rect(...xy.couple(box.topLeft(region)), ...xy.couple(box.dims(region)));
    ctx.fill();
    if (bordered) ctx.stroke();
  }

  textContainer(props: Draw2DTextContainerProps): void {
    const [dims, draw] = this.spacedTextDrawF(props);
    const { position } = props;
    this.container({
      region: box.construct(position, dims.width + 12, dims.height + 12),
      ...props,
    });
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
    const offset = Math.max(...textDims.map((td) => td[direction.dimension(d)])) + spacingPx;
    return [
      {
        [direction.dimension(direction.swap(d)) as "width"]: Math.max(
          ...textDims.map((td) => td[direction.dimension(direction.swap(d))])
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
}
