// Copyright 2025 Synnax Labs, Inc.
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
  clamp,
  color,
  type destructor,
  type scale,
  TimeRange,
  TimeSpan,
  xy,
} from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { flux } from "@/flux/aether";
import { type ranger as aetherRanger } from "@/ranger/aether";
import { status } from "@/status/aether";
import { synnax } from "@/synnax/aether";
import { theming } from "@/theming/aether";
import { Draw2D } from "@/vis/draw2d";
import { render } from "@/vis/render";

export const selectedStateZ = ranger.payloadZ.extend({
  viewport: bounds.bounds,
});

export type SelectedState = z.infer<typeof selectedStateZ>;

export const providerStateZ = z.object({
  cursor: xy.xy.or(z.null()),
  hovered: selectedStateZ.or(z.null()),
  count: z.number(),
});

interface InternalState {
  ranges: Map<string, ranger.Range>;
  client: Synnax | null;
  render: render.Context;
  requestRender: render.Requestor;
  draw: Draw2D;
  runAsync: status.ErrorHandler;
  removeListener: destructor.Destructor | null;
}

interface ProviderProps {
  dataToDecimalScale: scale.Scale;
  viewport: box.Box;
  region: box.Box;
  timeRange: TimeRange;
}

interface Store extends flux.Store {
  ranges: aetherRanger.FluxStore;
}

export class Provider extends aether.Leaf<typeof providerStateZ, InternalState> {
  static readonly TYPE = "range-provider";
  schema = providerStateZ;
  fetchedInitial: TimeRange = TimeRange.ZERO;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.render = render.Context.use(ctx);
    i.draw = new Draw2D(i.render.upper2d, theming.use(ctx));
    i.requestRender = render.useRequestor(ctx);
    i.runAsync = status.useErrorHandler(ctx);
    i.ranges ??= new Map();
    const client = synnax.use(ctx);
    i.requestRender("tool");
    if (client == null) return;
    i.client = client;
    const store = flux.useStore<Store>(ctx, this.key);
    i.removeListener?.();
    const removeOnSet = store.ranges.onSet((changed) => {
      if (i.client == null) return;
      if (color.isCrude(changed.color))
        i.ranges.set(changed.key, i.client.ranges.sugarOne(changed));
      this.setState((s) => ({ ...s, count: i.ranges.size }));
      i.requestRender("tool");
    });
    const removeOnDelete = store.ranges.onDelete(async (changed) => {
      i.ranges.delete(changed);
      this.setState((s) => ({ ...s, count: i.ranges.size }));
      i.requestRender("tool");
    });
    i.removeListener = () => {
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
      const ranges = await client.ranges.retrieve(timeRange);
      ranges.forEach((r) => {
        if (color.isCrude(r.color)) i.ranges.set(r.key, r);
      });
      this.setState((s) => ({ ...s, count: i.ranges.size }));
    }, "failed to fetch initial ranges");
  }

  render(props: ProviderProps): void {
    const { dataToDecimalScale, region, viewport, timeRange } = props;
    this.fetchInitial(timeRange);
    const { draw, ranges } = this.internal;
    const regionScale = dataToDecimalScale.scale(box.xBounds(region));
    const cursor = this.state.cursor == null ? null : this.state.cursor.x;
    let hoveredState: SelectedState | null = null;
    const clearScissor = draw.canvas.scissor(
      box.construct(
        { x: box.left(region), y: box.top(region) - 35 },
        { x: box.right(region), y: box.bottom(region) },
      ),
    );
    ranges.forEach((r) => {
      const cRes = color.colorZ.safeParse(r.color);
      if (!cRes.success) return;
      const c = cRes.data;
      let startPos = regionScale.pos(Number(r.timeRange.start.valueOf()));
      const endPos = regionScale.pos(Number(r.timeRange.end.valueOf()));
      if (endPos < box.left(region) || startPos > box.right(region)) return;
      startPos = clamp(startPos, box.left(region) - 2, box.right(region) - 1);
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
        backgroundColor: color.setAlpha(c, 0.2),
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
    clearScissor();
    if (hoveredState != null) this.setState((s) => ({ ...s, hovered: hoveredState }));
    else if (this.state.hovered) this.setState((s) => ({ ...s, hovered: null }));
  }
}
