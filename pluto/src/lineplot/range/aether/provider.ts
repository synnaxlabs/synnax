// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger, type Synnax } from "@synnaxlabs/client";
import { bounds, box, type scale, TimeRange, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { flux } from "@/flux/aether";
import {
  drawBars,
  drawFills,
  drawTooltip,
  hitTest,
  layout,
  stripHeight,
} from "@/lineplot/range/aether/strip";
import { ranger as aetherRanger } from "@/ranger/aether";
import { status } from "@/status/aether";
import { synnax } from "@/synnax/aether";
import { theming } from "@/theming/aether";
import { Draw2D } from "@/vis/draw2d";
import { render } from "@/vis/render";

export const selectedStateZ = ranger.payloadZ.extend({
  viewport: bounds.boundsZ(),
});

export type SelectedState = z.infer<typeof selectedStateZ>;

export const providerStateZ = z.object({
  cursor: xy.xyZ.or(z.null()),
  visible: z.boolean().optional().default(true),
  hovered: selectedStateZ.or(z.null()),
  count: z.number(),
  /** The strip's height in pixels, or 0 when no range is in view. */
  height: z.number().default(0),
});
export type ProviderState = z.infer<typeof providerStateZ>;

interface InternalState {
  retrieve: flux.Retrieve<aetherRanger.ListQuery, ranger.Range[]>;
  window?: TimeRange;
  client: Synnax | null;
  render: render.Context;
  requestRender: render.Requestor;
  draw: Draw2D;
  runAsync: status.ErrorHandler;
}

export interface ProviderProps {
  dataToDecimalScale: scale.Scale;
  viewport: box.Box;
  region: box.Box;
  timeRange: TimeRange;
}

/**
 * The most ranges the annotation strip draws for one viewport. A denser viewport draws
 * none: the strip cannot label them legibly, and the Core answers a limited query in
 * key order, so drawing a truncated answer would show an arbitrary subset that changes
 * as the viewport moves.
 */
const MAX_ANNOTATIONS = 100;

// The fetch window is the viewport padded by its span on each side, so a pan or a live
// plot's slide requeries at most once per viewport span.
const WINDOW_SPANS = 3n;

// One over the cap, so a saturated answer is distinguishable from a full one.
const FETCH_LIMIT = Number(WINDOW_SPANS) * MAX_ANNOTATIONS + 1;

const pad = (viewport: TimeRange): TimeRange =>
  new TimeRange(
    viewport.start.sub(viewport.span),
    viewport.end.add(viewport.span),
  ).boundBy(TimeRange.MAX);

// Zooming in 2x also requeries, so a saturated answer does not blank the strip at
// every deeper zoom.
const covers = (window: TimeRange, viewport: TimeRange): boolean =>
  window.contains(viewport) &&
  window.span.valueOf() <= 2n * WINDOW_SPANS * viewport.span.valueOf();

export class Provider extends aether.Leaf<typeof providerStateZ, InternalState> {
  static readonly TYPE = "range-provider";
  static readonly stateZ = providerStateZ;
  schema = providerStateZ;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.render = render.Context.use(ctx);
    i.draw = new Draw2D(i.render.upper2d, theming.use(ctx));
    i.requestRender = render.useRequestor(ctx);
    i.runAsync = status.useErrorHandler(ctx);
    i.retrieve ??= new flux.Retrieve({
      definition: aetherRanger.listDefinition,
      onChange: () => i.requestRender("tool"),
      onError: (error) => i.runAsync(error, "Failed to retrieve ranges"),
    });
    const client = synnax.use(ctx);
    if (client != null) i.client = client;
    i.requestRender("tool");
  }

  afterDelete(): void {
    this.internal.retrieve.close();
  }

  render(props: ProviderProps): void {
    const { dataToDecimalScale, region, viewport, timeRange } = props;
    const { internal: i } = this;
    if (i.window == null || !covers(i.window, timeRange)) i.window = pad(timeRange);
    if (i.client != null)
      i.retrieve.update(i.client, {
        where: { overlaps: i.window },
        limit: FETCH_LIMIT,
      });
    const { draw } = i;
    const fetched = i.retrieve.value ?? [];
    const start = timeRange.start.valueOf();
    const end = timeRange.end.valueOf();
    const inView = fetched.filter(
      (r) => r.timeRange.end.valueOf() >= start && r.timeRange.start.valueOf() <= end,
    );
    const saturated = fetched.length >= FETCH_LIMIT || inView.length > MAX_ANNOTATIONS;
    const regionScale = dataToDecimalScale.scale(box.xBounds(region));
    // Lanes pack over the axis bounds, not the viewport, so a range keeps its lane
    // while the viewport moves.
    const { bars: packed, lanes } = layout(
      saturated ? [] : inView,
      region,
      (v) => regionScale.pos(v),
      draw.theme,
    );
    const bars = packed.filter(
      (b) => b.end >= box.left(region) && b.start <= box.right(region),
    );
    const count = packed.length;
    const height = stripHeight(lanes);
    const { cursor, visible } = this.state;
    const hovered =
      visible && cursor != null ? hitTest(bars, region, lanes, cursor) : null;
    if (visible) {
      const clearScissor = draw.canvas.scissor(
        box.construct(
          { x: box.left(region) - 2, y: box.top(region) - height },
          { x: box.right(region) + 2, y: box.bottom(region) },
        ),
      );
      drawFills(draw, bars, region, hovered);
      drawBars(draw, bars, region, lanes, hovered);
      clearScissor();
      if (hovered != null && cursor != null) drawTooltip(draw, hovered, cursor, region);
    }
    const viewportScale = dataToDecimalScale.scale(box.xBounds(viewport));
    const r = hovered?.range;
    const hoveredState: SelectedState | null =
      r == null
        ? null
        : {
            key: r.key,
            parent: r.parent,
            name: r.name,
            color: r.color,
            labels: r.labels,
            timeRange: r.timeRange,
            hasChildren: r.hasChildren,
            viewport: {
              lower: viewportScale.pos(Number(r.timeRange.start.valueOf())),
              upper: viewportScale.pos(Number(r.timeRange.end.valueOf())),
            },
          };
    if (hoveredState != null) this.setState((s) => ({ ...s, hovered: hoveredState }));
    else if (this.state.hovered != null)
      this.setState((s) => ({ ...s, hovered: null }));
    if (this.state.count !== count || this.state.height !== height)
      this.setState((s) => ({ ...s, count, height }));
  }
}
