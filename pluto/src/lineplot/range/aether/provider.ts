// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger, type Synnax } from "@synnaxlabs/client";
import {
  bounds,
  box,
  color,
  type scale,
  TimeRange,
  TimeSpan,
  type TimeStamp,
  xy,
} from "@synnaxlabs/x";
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
  client: Synnax | null;
  render: render.Context;
  requestRender: render.Requestor;
  draw: Draw2D;
  runAsync: status.ErrorHandler;
}

interface ProviderProps {
  dataToDecimalScale: scale.Scale;
  viewport: box.Box;
  region: box.Box;
  timeRange: TimeRange;
}

// Widens fetch boundaries to whole minutes so panning does not requery on every
// viewport nudge.
const QUANTUM = TimeSpan.minutes(1);

const quantize = (timeRange: TimeRange): TimeRange => {
  const ceil = (ts: TimeStamp): TimeStamp =>
    ts.remainder(QUANTUM).isZero ? ts : ts.truncate(QUANTUM).add(QUANTUM);
  return new TimeRange(timeRange.start.truncate(QUANTUM), ceil(timeRange.end));
};

export class Provider extends aether.Leaf<typeof providerStateZ, InternalState> {
  static readonly TYPE = "range-provider";
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
        }, "failed to retrieve ranges"),
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
    if (i.client != null)
      i.retrieve.update(i.client, { overlapsWith: quantize(timeRange) });
    const { draw } = i;
    const ranges = i.retrieve.value ?? [];
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
      if (endPos < box.left(region) || startPos > box.right(region)) return;
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
