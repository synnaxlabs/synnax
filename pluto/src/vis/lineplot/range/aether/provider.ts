// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger, type signals, type Synnax } from "@synnaxlabs/client";
import { bounds, box, clamp, type scale, TimeRange, TimeSpan, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { color } from "@/color/core";
import { aether } from "@/ether";
import { synnax } from "@/synnax/aether";
import { theming } from "@/theming/aether";
import { Draw2D } from "@/vis/draw2d";
import { render } from "@/vis/render";

export const selectedStateZ = ranger.payloadZ.extend({
  viewport: bounds.bounds,
});

export type SelectedState = z.infer<typeof selectedStateZ>;

const hasColor = (c?: string): boolean => color.Color.z.safeParse(c).success;

export const providerStateZ = z.object({
  cursor: xy.xy.or(z.null()),
  hovered: selectedStateZ.or(z.null()),
  count: z.number(),
});

interface InternalState {
  ranges: Map<string, ranger.Range>;
  client: Synnax | null;
  render: render.Context;
  draw: Draw2D;
  tracker: signals.Observable<string, ranger.Range>;
}

interface AnnotationProps {
  dataToDecimalScale: scale.Scale;
  viewport: box.Box;
  region: box.Box;
  timeRange: TimeRange;
}

export class Provider extends aether.Leaf<typeof providerStateZ, InternalState> {
  static readonly TYPE = "range-provider";
  schema = providerStateZ;
  fetchedInitial: TimeRange = TimeRange.ZERO;

  async afterUpdate(): Promise<void> {
    const { internal: i } = this;
    i.render = render.Context.use(this.ctx);
    i.draw = new Draw2D(i.render.upper2d, theming.use(this.ctx));

    i.ranges ??= new Map();
    const client = synnax.use(this.ctx);
    if (client == null) return;
    i.client = client;

    if (i.tracker != null) return;
    i.tracker = await i.client.ranges.openTracker();
    i.tracker.onChange(async (c) => {
      c.forEach(async (r) => {
        if (r.variant === "delete") i.ranges.delete(r.key);
        else if (hasColor(r.value.color)) i.ranges.set(r.key, r.value);
      });
      render.Controller.requestRender(this.ctx, render.REASON_TOOL);
      this.setState((s) => ({ ...s, count: i.ranges.size }));
    });
    render.Controller.requestRender(this.ctx, render.REASON_TOOL);
  }

  private async fetchInitial(timeRange: TimeRange): Promise<void> {
    const { internal: i } = this;
    if (
      i.client == null ||
      this.fetchedInitial.roughlyEquals(timeRange, TimeSpan.minutes(1))
    )
      return;
    this.fetchedInitial = timeRange;
    const ranges = await i.client.ranges.retrieve(timeRange);
    ranges.forEach((r) => {
      if (hasColor(r.color)) i.ranges.set(r.key, r);
    });
    this.setState((s) => ({ ...s, count: i.ranges.size }));
  }

  async render(props: AnnotationProps): Promise<void> {
    const { dataToDecimalScale, region, viewport, timeRange } = props;
    await this.fetchInitial(timeRange);
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
      const cRes = color.Color.z.safeParse(r.color);
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
          name: r.name,
          color: r.color,
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
        backgroundColor: c.setAlpha(0.2),
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
            ? c.setAlpha(0.4)
            : (t) => (hovered ? t.colors.gray.l2 : t.colors.gray.l0),
        bordered: true,
        borderWidth: 1,
        borderRadius: 2,
        borderColor: c.setAlpha(0.8),
      });
      draw.text({
        text: r.name,
        position: { x: startPos + 8, y: box.top(region) - 30 },
        level: "small",
        shade: 8,
        weight: 500,
        maxWidth: endPos - startPos - 16,
      });
    });
    clearScissor();
    if (hoveredState != null) this.setState((s) => ({ ...s, hovered: hoveredState }));
    else if (this.state.hovered) this.setState((s) => ({ ...s, hovered: null }));
  }
}
