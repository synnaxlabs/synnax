// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { annotation, ontology, type Synnax } from "@synnaxlabs/client";
import {
  bounds,
  box,
  color,
  type Destructor,
  scale,
  TimeRange,
  TimeSpan,
  xy,
} from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { flux } from "@/flux/aether";
import { status } from "@/status/aether";
import { synnax } from "@/synnax/aether";
import { theming } from "@/theming/aether";
import { Draw2D } from "@/vis/draw2d";
import { render } from "@/vis/render";

const COMMENT_BOX_OFFSET: xy.XY = xy.construct(8, 8);
const COMMENT_BOX_PADDING: xy.XY = xy.construct(8, 6);
const COMMENT_LINE_HEIGHT: number = 16;
const COMMENT_MAX_WIDTH: number = 200;
const COMMENT_CHARS_PER_LINE: number = 25;

export const selectedStateZ = annotation.annotationZ.extend({
  viewport: bounds.bounds,
});

export type SelectedState = z.infer<typeof selectedStateZ>;

export const providerStateZ = z.object({
  cursor: xy.xy.or(z.null()),
  hovered: selectedStateZ.or(z.null()),
  count: z.number(),
  parents: ontology.idZ.array(),
});

export type ProviderState = z.infer<typeof providerStateZ>;

interface InternalState {
  annotations: Map<string, annotation.Annotation>;
  client: Synnax | null;
  render: render.Context;
  requestRender: render.Requestor;
  draw: Draw2D;
  runAsync: status.ErrorHandler;
  removeListeners: Destructor | null;
}

interface ProviderProps {
  dataToDecimalScale: scale.Scale;
  viewport: box.Box;
  region: box.Box;
  timeRange: TimeRange;
}

export class Provider extends aether.Leaf<typeof providerStateZ, InternalState> {
  static readonly TYPE = "annotation-provider";
  schema = providerStateZ;
  fetchedInitial: TimeRange = TimeRange.ZERO;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.render = render.Context.use(ctx);
    i.draw = new Draw2D(i.render.upper2d, theming.use(ctx));
    i.requestRender = render.useRequestor(ctx);
    i.runAsync = status.useErrorHandler(ctx);
    i.annotations ??= new Map();
    const client = synnax.use(ctx);
    i.requestRender("tool");
    if (client == null) return;
    i.client = client;

    i.removeListeners = flux.useListener(
      ctx,
      [
        {
          channel: annotation.SET_CHANNEL_NAME,
          onChange: flux.parsedHandler(annotation.annotationZ, async ({ changed }) => {
            i.annotations.set(changed.key, changed);
            this.setState((s) => ({ ...s, count: i.annotations.size }));
            i.requestRender("tool");
          }),
        },
        {
          channel: annotation.DELETE_CHANNEL_NAME,
          onChange: flux.parsedHandler(annotation.keyZ, async ({ changed }) => {
            i.annotations.delete(changed);
            this.setState((s) => ({ ...s, count: i.annotations.size }));
            i.requestRender("tool");
          }),
        },
      ],
      i.removeListeners,
    );
  }

  private fetchInitial(timeRange: TimeRange): void {
    const { internal: i } = this;
    const { client, runAsync } = i;
    if (client == null || this.fetchedInitial.equals(timeRange, TimeSpan.minutes(1)))
      return;

    this.fetchedInitial = timeRange;
    runAsync(async () => {
      const children = await client.ontology.retrieveChildren(this.state.parents, {
        types: [annotation.ONTOLOGY_TYPE],
      });
      const annotations = await client.annotations.retrieve({
        keys: children.map((c) => c.id.key),
      });
      annotations.forEach((a) => {
        i.annotations.set(a.key, a);
      });
      this.setState((s) => ({ ...s, count: i.annotations.size }));
    }, "failed to fetch initial annotations");
  }

  render(props: ProviderProps): void {
    const { dataToDecimalScale, region, viewport, timeRange } = props;
    this.fetchInitial(timeRange);
    const { draw, annotations } = this.internal;
    const regionScale = dataToDecimalScale.scale(box.xBounds(region));
    const reverseScale = scale.XY.scale(region).scale(box.DECIMAL);
    const cursor = this.state.cursor == null ? null : this.state.cursor.x;
    let hoveredState: SelectedState | null = null;
    const clearScissor = draw.canvas.scissor(
      box.construct(
        { x: box.left(region), y: box.top(region) - 35 },
        { x: box.right(region), y: box.bottom(region) },
      ),
    );

    annotations.forEach((a) => {
      const startTime = Number(a.timeRange.start.valueOf());
      const position = regionScale.pos(startTime);

      if (position < box.left(region) || position > box.right(region)) return;

      const annotationColor = color.construct("#3774d0");
      const markerRadius = 4;

      let hovered = false;
      if (cursor != null)
        hovered = bounds.contains(
          { lower: position - markerRadius, upper: position + markerRadius },
          cursor,
        );

      if (hovered)
        hoveredState = {
          ...a,
          viewport: {
            lower: dataToDecimalScale.scale(box.xBounds(viewport)).pos(startTime),
            upper: dataToDecimalScale.scale(box.xBounds(viewport)).pos(startTime),
          },
        };

      const markerTop = box.top(region);
      const markerBottom = box.bottom(region);

      draw.line({
        start: xy.construct(position, markerTop),
        end: xy.construct(position, markerBottom - markerRadius),
        stroke: color.setAlpha(annotationColor, hovered ? 1.0 : 0.6),
        lineWidth: hovered ? 2 : 1,
        lineDash: 0,
      });

      draw.circle({
        position: xy.construct(position, markerBottom),
        radius: markerRadius,
        fill: hovered ? annotationColor : color.setAlpha(annotationColor, 0.8),
      });

      const markerPosition = xy.construct(position, markerTop);
      const relativePosition = reverseScale.pos(markerPosition);

      const messageLines = this.wrapText(a.message, COMMENT_CHARS_PER_LINE);
      const boxWidth = Math.min(
        Math.max(
          messageLines.reduce((max, line) => Math.max(max, line.length), 0) * 8 + 16,
          120,
        ),
        COMMENT_MAX_WIDTH,
      );
      const boxHeight = messageLines.length * COMMENT_LINE_HEIGHT + 12;

      let commentX = position + COMMENT_BOX_OFFSET.x;
      let commentY = markerTop + COMMENT_BOX_OFFSET.y + boxHeight;

      if (relativePosition.x > 0.7)
        commentX = position - COMMENT_BOX_OFFSET.x - boxWidth;

      if (relativePosition.y < 0.3) commentY = markerTop + COMMENT_BOX_OFFSET.y;

      const commentRegion = box.construct(
        { x: commentX, y: commentY },
        { width: boxWidth, height: boxHeight },
      );

      draw.container({
        region: commentRegion,
        backgroundColor: (t) => t.colors.gray.l0,
        bordered: true,
        borderWidth: 1,
        borderRadius: 4,
        borderColor: (t) => t.colors.border,
      });

      messageLines.forEach((line, i) => {
        draw.text({
          position: xy.construct(
            commentX + COMMENT_BOX_PADDING.x,
            commentY + COMMENT_BOX_PADDING.y + i * COMMENT_LINE_HEIGHT,
          ),
          text: line,
          level: "small",
          weight: 400,
          maxWidth: boxWidth - 16,
        });
      });
    });

    clearScissor();
    if (hoveredState != null) this.setState((s) => ({ ...s, hovered: hoveredState }));
    else if (this.state.hovered) this.setState((s) => ({ ...s, hovered: null }));
  }

  private wrapText(text: string, maxLength: number): string[] {
    if (text.length <= maxLength) return [text];

    const words = text.split(" ");
    const lines: string[] = [];
    let currentLine = "";

    for (const word of words)
      if (`${currentLine} ${word}`.length <= maxLength)
        currentLine = currentLine ? `${currentLine} ${word}` : word;
      else {
        if (currentLine) lines.push(currentLine);
        currentLine = word;
      }

    if (currentLine) lines.push(currentLine);
    return lines;
  }
}
