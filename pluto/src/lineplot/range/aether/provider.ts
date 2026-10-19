// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger, type Synnax } from "@synnaxlabs/client";
import { bounds, box, color, type scale, TimeRange, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { flux } from "@/flux/aether";
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
 * none: the strip cannot label them legibly in 32 pixels, and the Core answers a
 * limited query in key order, so drawing a truncated answer would show an arbitrary
 * subset that changes as the viewport moves.
 */
export const MAX_ANNOTATIONS = 100;

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
      onError: (error) =>
        i.runAsync(async () => {
          throw error;
        }, "Failed to retrieve ranges"),
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
      i.retrieve.update(i.client, { overlapsWith: i.window, limit: FETCH_LIMIT });
    const { draw } = i;
    const fetched = i.retrieve.value ?? [];
    const start = timeRange.start.valueOf();
    const end = timeRange.end.valueOf();
    const inView = fetched.filter(
      (r) => r.timeRange.end.valueOf() >= start && r.timeRange.start.valueOf() <= end,
    );
    const ranges =
      fetched.length >= FETCH_LIMIT || inView.length > MAX_ANNOTATIONS ? [] : inView;
    const visible = this.state.visible !== false;
    const regionScale = dataToDecimalScale.scale(box.xBounds(region));
    const cursor = this.state.cursor == null ? null : this.state.cursor.x;
    let hoveredState: SelectedState | null = null;
    let visibleCount = 0;
    const clearScissor = visible
      ? draw.canvas.scissor(
          box.construct(
            { x: box.left(region), y: box.top(region) - 35 },
            { x: box.right(region), y: box.bottom(region) },
          ),
        )
      : null;
    ranges.forEach((r) => {
      const cRes = color.colorZ.safeParse(r.color);
      if (!cRes.success) return;
      const c = cRes.data;
      let startPos = regionScale.pos(Number(r.timeRange.start.valueOf()));
      const endPos = regionScale.pos(Number(r.timeRange.end.valueOf()));
      visibleCount++;
      if (!visible) return;
      startPos = bounds.clamp(
        { lower: box.left(region) - 2, upper: box.right(region) - 1 },
        startPos,
      );
      let hovered = false;
      if (cursor != null)
        hovered = bounds.contains({ lower: startPos, upper: endPos }, cursor);
      if (hovered)
        hoveredState = {
          key: r.key,
          parent: r.parent,
          name: r.name,
          color: r.color,
          labels: r.labels,
          timeRange: r.timeRange,
          viewport: {
            lower: dataToDecimalScale
              .scale(box.xBounds(viewport))
              .pos(Number(r.timeRange.start.valueOf())),
            upper: dataToDecimalScale
              .scale(box.xBounds(viewport))
              .pos(Number(r.timeRange.end.valueOf())),
          },
        };
      draw.container({
        region: box.construct(
          { x: startPos, y: box.top(region) - 1 },
          { x: endPos, y: box.bottom(region) - 1 },
        ),
        backgroundColor: color.setAlpha(c, 0.1),
        bordered: false,
      });
      const titleRegion = box.construct(
        { x: startPos + 1, y: box.top(region) - 34 },
        { x: endPos - 1, y: box.top(region) - 12 },
      );
      draw.container({
        region: titleRegion,
        backgroundColor:
          box.width(titleRegion) < 20
            ? color.setAlpha(c, 0.4)
            : (t) => (hovered ? t.colors.gray.l2 : t.colors.gray.l0),
        bordered: true,
        borderWidth: 1,
        borderRadius: 2,
        borderColor: color.setAlpha(c, 0.8),
      });
      draw.text({
        text: r.name,
        position: { x: startPos + 8, y: box.top(region) - 30 },
        level: "small",
        shade: 10,
        weight: 500,
        maxWidth: endPos - startPos - 16,
      });
    });
    clearScissor?.();
    if (hoveredState != null) this.setState((s) => ({ ...s, hovered: hoveredState }));
    else if (this.state.hovered) this.setState((s) => ({ ...s, hovered: null }));
    if (this.state.count !== visibleCount)
      this.setState((s) => ({ ...s, count: visibleCount }));
  }
}
