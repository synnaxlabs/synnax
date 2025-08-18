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
  TimeStamp,
  xy,
} from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { type annotation as aetherAnnotation } from "@/annotation/aether";
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
const ANNOTATION_MIN_SPACING: number = 10;

interface AnnotationLayout {
  region: box.Box;
  position: number;
  markerPosition: xy.XY;
}

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

    const store = flux.useClient<aetherAnnotation.SubStore>(ctx, this.key);
    i.removeListeners?.();

    const removeOnSet = store.annotations.onSet(async (changed) => {
      if (i.client == null) return;
      i.annotations.set(
        changed.key,
        await i.client.annotations.retrieve({
          key: changed.key,
          includeCreator: true,
        }),
      );
      this.setState((s) => ({ ...s, count: i.annotations.size }));
      i.requestRender("tool");
    });

    const removeOnDelete = store.annotations.onDelete(async (key) => {
      i.annotations.delete(key);
      this.setState((s) => ({ ...s, count: i.annotations.size }));
      i.requestRender("tool");
    });

    i.removeListeners = () => {
      removeOnSet();
      removeOnDelete();
    };
  }

  private fetchInitial(timeRange: TimeRange): void {
    const { internal: i } = this;
    const { client, runAsync } = i;
    if (client == null || this.fetchedInitial.equals(timeRange, TimeSpan.minutes(1)))
      return;

    this.fetchedInitial = timeRange;
    runAsync(async () => {
      const children = await client.ontology.retrieveChildren(this.state.parents, {
        types: ["annotation"],
      });
      const annotations = await client.annotations.retrieve({
        keys: children.map((c) => c.id.key),
        includeCreator: true,
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

    // Track occupied regions for collision avoidance
    const occupiedLayouts: AnnotationLayout[] = [];

    // Sort annotations by time to ensure consistent collision resolution
    const sortedAnnotations = Array.from(annotations.values()).sort(
      (a, b) =>
        Number(a.timeRange.start.valueOf()) - Number(b.timeRange.start.valueOf()),
    );

    sortedAnnotations.forEach((a) => {
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

      // Create annotation content with timing and creator info
      const annotationContent = this.createAnnotationContent(a);
      const messageLines = this.wrapText(annotationContent, COMMENT_CHARS_PER_LINE);

      const boxWidth = Math.min(
        Math.max(
          messageLines.reduce((max, line) => Math.max(max, line.length), 0) * 8 + 16,
          140,
        ),
        COMMENT_MAX_WIDTH,
      );
      const boxHeight = messageLines.length * COMMENT_LINE_HEIGHT + 12;

      let commentX = position + COMMENT_BOX_OFFSET.x;
      let commentY = markerTop + COMMENT_BOX_OFFSET.y + boxHeight;

      if (relativePosition.x > 0.7)
        commentX = position - COMMENT_BOX_OFFSET.x - boxWidth;

      if (relativePosition.y < 0.3) commentY = markerTop + COMMENT_BOX_OFFSET.y;

      // Create initial comment region
      let commentRegion = box.construct(
        { x: commentX, y: commentY },
        { width: boxWidth, height: boxHeight },
      );

      // Apply collision avoidance
      commentRegion = this.avoidCollisions(commentRegion, occupiedLayouts, region);
      commentX = box.left(commentRegion);
      commentY = box.top(commentRegion);

      // Add this layout to occupied regions
      occupiedLayouts.push({
        region: commentRegion,
        position,
        markerPosition,
      });

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

  private createAnnotationContent(annotation: annotation.Annotation): string {
    const timestamp = new TimeStamp(annotation.timeRange.start);
    const timeStr = timestamp.fString("time");

    let creatorStr = "";
    if (annotation.creator) {
      const { firstName, lastName, username } = annotation.creator;
      const displayName = firstName && lastName ? `${firstName} ${lastName}` : username;
      creatorStr = ` - ${displayName}`;
    }

    return `${timeStr}${creatorStr}\n${annotation.message}`;
  }

  private avoidCollisions(
    targetRegion: box.Box,
    occupiedLayouts: AnnotationLayout[],
    plotRegion: box.Box,
  ): box.Box {
    let adjustedRegion = targetRegion;
    let attempts = 0;
    const maxAttempts = 10;

    while (attempts < maxAttempts) {
      let hasCollision = false;

      for (const layout of occupiedLayouts)
        if (this.regionsOverlap(adjustedRegion, layout.region)) {
          hasCollision = true;

          // Try to move the box vertically to avoid collision
          const moveUp =
            box.top(layout.region) -
            box.height(adjustedRegion) -
            ANNOTATION_MIN_SPACING;
          const moveDown = box.bottom(layout.region) + ANNOTATION_MIN_SPACING;

          // Choose the direction that keeps the annotation more visible
          const currentY = box.top(adjustedRegion);
          const upDistance = Math.abs(currentY - moveUp);
          const downDistance = Math.abs(currentY - moveDown);

          let newY = upDistance < downDistance ? moveUp : moveDown;

          // Ensure the annotation stays within the plot region bounds
          const topBound = box.top(plotRegion) - 100; // Allow some overlap above
          const bottomBound = box.bottom(plotRegion) - box.height(adjustedRegion);

          newY = Math.max(topBound, Math.min(newY, bottomBound));

          adjustedRegion = box.construct(
            { x: box.left(adjustedRegion), y: newY },
            { width: box.width(adjustedRegion), height: box.height(adjustedRegion) },
          );

          break;
        }

      if (!hasCollision) break;
      attempts++;
    }

    return adjustedRegion;
  }

  private regionsOverlap(region1: box.Box, region2: box.Box): boolean {
    const margin = ANNOTATION_MIN_SPACING;

    return !(
      box.right(region1) + margin < box.left(region2) ||
      box.left(region1) > box.right(region2) + margin ||
      box.bottom(region1) + margin < box.top(region2) ||
      box.top(region1) > box.bottom(region2) + margin
    );
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
