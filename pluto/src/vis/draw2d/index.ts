// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box, Dimensions, Direction, XY } from "@synnaxlabs/x";

import { Color } from "@/color";
import { dimensions } from "@/text/dimensions";
import { Level } from "@/text/types";
import { fontString } from "@/theming/core/fontString";
import { Theme } from "@/theming/core/theme";

export interface Draw2DLineProps {
  stroke: Color;
  lineWidth: number;
  lineDash: number;
  start: XY;
  end: XY;
}

export interface Draw2DRuleProps extends Omit<Draw2DLineProps, "start" | "end"> {
  direction: Direction;
  region: Box;
  position: number;
}

export interface Draw2DCircleProps {
  fill: Color;
  radius: number;
  position: XY;
}

export interface Draw2DContainerProps {
  region: Box;
  bordered?: boolean;
  rounded?: boolean;
  borderColor?: Color;
  borderRadius?: number;
  borderWidth?: number;
  backgroundColor?: Color;
}

export interface DrawTextProps {
  text: string;
  position: XY;
  level: Level;
  direction: Direction;
}

export interface Draw2DMeasureTextContainerProps {
  text: string[];
  direction: Direction;
  level: Level;
  spacing?: number;
}

export interface Draw2DTextContainerProps
  extends Omit<Draw2DContainerProps, "region">,
    Draw2DMeasureTextContainerProps {
  position: XY;
}

export class Draw2D {
  readonly canvas: OffscreenCanvasRenderingContext2D;
  readonly theme: Theme;

  constructor(canvas: OffscreenCanvasRenderingContext2D, theme: Theme) {
    this.canvas = canvas;
    this.theme = theme;
  }

  rule({ direction, region, position, ...props }: Draw2DRuleProps): void {
    if (direction.isX)
      return this.line({
        start: new XY(region.left, position),
        end: new XY(region.right, position),
        ...props,
      });
    return this.line({
      start: new XY(position, region.top),
      end: new XY(position, region.bottom),
      ...props,
    });
  }

  line({ stroke, lineWidth, lineDash, start, end }: Draw2DLineProps): void {
    const ctx = this.canvas;
    ctx.strokeStyle = stroke.hex;
    ctx.lineWidth = lineWidth;
    ctx.setLineDash([lineDash]);
    ctx.beginPath();
    ctx.moveTo(...start.couple);
    ctx.lineTo(...end.couple);
    ctx.stroke();
  }

  circle({ fill, radius, position }: Draw2DCircleProps): void {
    const ctx = this.canvas;
    ctx.fillStyle = fill.hex;
    ctx.beginPath();
    ctx.arc(...position.couple, radius, 0, 2 * Math.PI);
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
      ctx.roundRect(...region.topLeft.couple, ...region.dims.couple, borderRadius);
    else ctx.rect(...region.topLeft.couple, ...region.dims.couple);
    ctx.fill();
    if (bordered) ctx.stroke();
  }

  textContainer(props: Draw2DTextContainerProps): void {
    const [dims, draw] = this.spacedTextDrawF(props);
    const { position } = props;
    this.container({
      region: new Box(position, dims.width + 12, dims.height + 12),
      ...props,
    });
    draw(position.translate([6, 6]));
  }

  spacedTextDrawF({
    text,
    direction,
    spacing = 1,
    level = "p",
  }: Draw2DMeasureTextContainerProps): [Dimensions, (base: XY) => void] {
    const font = fontString(this.theme, level);
    const textDims = text.map((t) => dimensions(t, font, this.canvas));
    const spacingPx = this.theme.sizes.base * spacing;
    const offset = Math.max(...textDims.map((d) => d[direction.dimension])) + spacingPx;
    return [
      new Dimensions({
        [direction.inverse.dimension as "width"]: Math.max(
          ...textDims.map((d) => d[direction.inverse.dimension])
        ),
        [direction.dimension as "height"]: offset * text.length - spacingPx,
      }),

      (position: XY) => {
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
