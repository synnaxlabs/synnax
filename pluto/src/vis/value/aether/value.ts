// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, notation } from "@synnaxlabs/x";
import { box, location, scale, xy } from "@synnaxlabs/x/spatial";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { telem } from "@/telem/aether";
import { noopColorSourceSpec } from "@/telem/aether/noop";
import { text } from "@/text/core";
import { theming } from "@/theming/aether";
import { type Element } from "@/vis/diagram/aether/Diagram";
import { type FillTextOptions } from "@/vis/draw2d/canvas";
import { render } from "@/vis/render";

const FILL_TEXT_OPTIONS: FillTextOptions = { useAtlas: true };

const valueState = z.object({
  box: box.box,
  telem: telem.stringSourceSpecZ.optional().default(telem.noopStringSourceSpec),
  backgroundTelem: telem.colorSourceSpecZ.optional().default(telem.noopColorSourceSpec),
  level: text.levelZ.optional().default("p"),
  color: color.colorZ.optional().default(color.ZERO),
  precision: z.number().optional().default(2),
  staleness_timeout: z.number().optional().default(5),
  staleness_color: color.colorZ.optional().default(color.ZERO),
  minWidth: z.number().optional().default(60),
  width: z.number().optional(),
  notation: notation.notationZ.optional().default("standard"),
  location: location.xy.optional().default({ x: "left", y: "center" }),
  useWidthForBackground: z.boolean().optional().default(false),
  valueBackgroundShift: xy.xy.optional().default(xy.ZERO),
  valueBackgroundOverScan: xy.xy.optional().default(xy.ZERO),
});

const CANVAS_VARIANTS: render.Canvas2DVariant[] = ["upper2d", "lower2d"];
const DATA_COUNT_THRESHOLD = 2; // Number of packets before non-stale color is shown

export interface ValueProps {
  scale?: scale.XY;
}

interface InternalState {
  theme: theming.Theme;
  renderCtx: render.Context;
  telem: telem.StringSource;
  stopListening?: () => void;
  backgroundTelem: telem.ColorSource;
  stopListeningBackground?: () => void;
  requestRender: render.Requestor | null;
  textColor: color.Color;
  fontString: string;
  staleTimeout?: ReturnType<typeof setTimeout>;
  isInitialized: boolean;
  timed_out: boolean;
  dataCount: number;
}

export class Value
  extends aether.Leaf<typeof valueState, InternalState>
  implements Element
{
  static readonly TYPE = "value";
  static readonly z = valueState;
  schema = Value.z;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.renderCtx = render.Context.use(ctx);
    i.theme = theming.use(ctx);

    i.textColor = this.state.staleness_color;
    
    if (i.isInitialized === undefined) {
      i.timed_out = true;
      i.isInitialized = true;
      i.dataCount = 0;
    }
    
    if (this.state.telem.type === "source-pipeline") {
      const pipelineProps = telem.sourcePipelinePropsZ.parse(this.state.telem.props);
      const stringifierSegment = pipelineProps.segments.stringifier;
      if (stringifierSegment?.type === "stringify-number") {
        const stringifierProps = telem.stringifyNumberProps.parse(stringifierSegment.props);
        if (stringifierProps.staleness_timeout !== this.state.staleness_timeout){
          this.setState(prev => ({ ...prev, staleness_timeout: stringifierProps.staleness_timeout }));
          i.dataCount = 0;
        }
        if (stringifierProps.staleness_color !== this.state.staleness_color) {
          this.setState(prev => ({ ...prev, staleness_color: stringifierProps.staleness_color }));
          i.textColor = stringifierProps.staleness_color;
          i.dataCount = 0;
        }
      }
    }

    i.telem = telem.useSource(ctx, this.state.telem, i.telem);
    i.stopListening?.();
    i.stopListening = i.telem.onChange(() => {
      // Only increment data count until we reach the threshold
      if (i.dataCount >= DATA_COUNT_THRESHOLD) {
        i.timed_out = false;
        if (color.isZero(this.state.color)) i.textColor = i.theme.colors.gray.l10;
        else i.textColor = this.state.color;
      } 
      else i.dataCount++;

      this.resetStaleTimeout();
      this.requestRender();
    });
    i.fontString = theming.fontString(i.theme, { level: this.state.level, code: true });
    i.backgroundTelem = telem.useSource(
      ctx,
      this.state.backgroundTelem,
      i.backgroundTelem,
    );
    i.stopListeningBackground?.();
    i.stopListeningBackground = i.backgroundTelem.onChange(() => this.requestRender());
    i.requestRender = render.useOptionalRequestor(ctx);
    this.requestRender();
  }

  afterDelete(): void {
    const { internal: i } = this;
    i.stopListening?.();
    i.stopListeningBackground?.();
    if (i.staleTimeout) clearTimeout(i.staleTimeout);
    i.telem.cleanup?.();
    i.backgroundTelem.cleanup?.();
    if (i.requestRender == null)
      i.renderCtx.erase(box.construct(this.state.box), xy.ZERO, ...CANVAS_VARIANTS);
    else i.requestRender("layout");
  }

  private resetStaleTimeout(): void {
    const { internal: i } = this;
    if (i.staleTimeout) clearTimeout(i.staleTimeout);
    // Set timeout to return to staleness color after specified duration
    const timeoutDuration = this.state.staleness_timeout * 1000;
    i.staleTimeout = setTimeout(() => {
      i.textColor = this.state.staleness_color;
      i.timed_out = true;
      this.requestRender();
    }, timeoutDuration);
  }

  private requestRender(): void {
    const { requestRender } = this.internal;
    if (requestRender != null) requestRender("layout");
    else void this.render({});
  }

  private get fontHeight(): number {
    const { theme } = this.internal;
    return theme.typography[this.state.level].size * theme.sizes.base;
  }

  private maybeUpdateWidth(width: number) {
    const { theme } = this.internal;
    const requiredWidth = width + theme.sizes.base + this.fontHeight;
    if (
      this.state.width == null ||
      this.state.width + this.fontHeight * 0.5 < requiredWidth ||
      (this.state.minWidth > requiredWidth && this.state.width !== this.state.minWidth)
    )
      this.setState((p) => ({ ...p, width: Math.max(requiredWidth, p.minWidth) }));
    else if (this.state.width - this.fontHeight > requiredWidth)
      this.setState((p) => ({ ...p, width: Math.max(requiredWidth, p.minWidth) }));
  }

  render({ viewportScale = scale.XY.IDENTITY }): void {
    const { renderCtx, telem, backgroundTelem, fontString, requestRender } =
      this.internal;
    const { location, box: b } = this.state;
    if (box.areaIsZero(b)) return;
    const bTopLeft = box.topLeft(b);
    const bWidth = box.width(b);
    const bHeight = box.height(b);
    const canvas = renderCtx.lower2d.applyScale(viewportScale);
    let value = telem.value();
    canvas.font = fontString;
    const fontHeight = this.fontHeight;
    const isNegative = value[0] == "-";
    if (isNegative) value = value.slice(1);

    const { theme } = this.internal;
    const dims = canvas.textDimensions(value, FILL_TEXT_OPTIONS);
    const width = dims.width + theme.sizes.base;
    const height = dims.height;
    if (requestRender == null) renderCtx.erase(box.construct(this.prevState.box));

    this.maybeUpdateWidth(width);
    const labelOffset = { ...xy.ZERO };
    if (location.x === "left") labelOffset.x = 6 + fontHeight * 0.75;
    else if (location.x === "center") labelOffset.x = bWidth / 2 - width / 2;
    if (location.y === "center") labelOffset.y = bHeight / 2 + height / 2;

    const labelPosition = xy.translate(bTopLeft, labelOffset);

    let setDefaultFillStyle = true;
    if (this.state.backgroundTelem.type != noopColorSourceSpec.type) {
      const colorValue = backgroundTelem.value();
      const isZero = color.isZero(colorValue);
      if (!isZero) {
        setDefaultFillStyle = false;
        canvas.fillStyle = color.hex(colorValue);
        const width = this.state.useWidthForBackground
          ? (this.state.width ?? this.state.minWidth)
          : box.width(b);
        canvas.fillRect(
          ...xy.couple(xy.translate(bTopLeft, this.state.valueBackgroundShift)),
          width + this.state.valueBackgroundOverScan.x,
          bHeight + this.state.valueBackgroundOverScan.y,
        );
        const textColor = color.pickByContrast(
          colorValue,
          theme.colors.gray.l0,
          theme.colors.gray.l11,
        );
        canvas.fillStyle = color.hex(textColor);
      }
    }
    if (setDefaultFillStyle)
      // Show normal color only after multiple data packets AND not timed out
      if (this.internal.dataCount >= DATA_COUNT_THRESHOLD && !this.internal.timed_out)
        if (color.isZero(this.state.color)) canvas.fillStyle = color.hex(theme.colors.gray.l10);
        else canvas.fillStyle = color.hex(this.state.color);
      else
        canvas.fillStyle = color.hex(this.state.staleness_color);

    // If the value is negative, chop of the negative sign and draw it separately
    // so that the first digit always stays in the same position, regardless of the sign.
    if (isNegative)
      canvas.fillText(
        "-",
        // 0.55 is a multiplier of the font height that seems to keep the sign in
        // the right place.
        ...xy.couple(xy.translateX(labelPosition, -fontHeight * 0.6)),
        undefined,
        FILL_TEXT_OPTIONS,
      );
    canvas.fillText(value, ...xy.couple(labelPosition), undefined, FILL_TEXT_OPTIONS);
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Value.TYPE]: Value,
};

