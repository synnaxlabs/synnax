// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color, ranger, type Synnax } from "@synnaxlabs/client";
import { bounds, box, type scale, TimeRange, TimeStamp, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { color as aetherColor } from "@/color/aether";
import { flux } from "@/flux/aether";
import {
  type Bar,
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
  /** The menu of the hovered range is open, so its tooltip hides. */
  menuOpen: z.boolean().default(false),
  hovered: selectedStateZ.or(z.null()),
  count: z.number(),
  /** The strip's height in pixels, or 0 when no range is in view. */
  height: z.number().default(0),
});
export type ProviderState = z.infer<typeof providerStateZ>;

interface InternalState {
  retrieve: flux.Retrieve<aetherRanger.ListQuery, ranger.Range[]>;
  colors: flux.Retrieve<aetherColor.ListQuery, color.Named[]>;
  named: Map<color.Key, color.Named>;
  window?: TimeRange;
  client: Synnax | null;
  render: render.Context;
  requestRender: render.Requestor;
  draw: Draw2D;
  runAsync: status.ErrorHandler;
  // Draws the strip again when it next changes on its own.
  timer?: ReturnType<typeof setTimeout>;
}

export interface ProviderProps {
  dataToDecimalScale: scale.Scale;
  viewport: box.Box;
  region: box.Box;
  timeRange: TimeRange;
  /** The newest sample, when the x axis ends at it. */
  newest?: TimeStamp;
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

// Redraw delays in milliseconds: each second for T+, and 15 Hz while a head pulses. A
// faster pulse repaints the whole canvas more often for no visible gain.
const CLOCK_PERIOD = 1000;
const HEAD_PERIOD = 66;
// The longest delay setTimeout accepts. It fires a longer one at once.
const MAX_DELAY = 2 ** 31 - 1;

export class Provider extends aether.Leaf<typeof providerStateZ, InternalState> {
  static readonly TYPE = "range-provider";
  static readonly stateZ = providerStateZ;
  override schema = providerStateZ;

  override afterUpdate(ctx: aether.Context): void {
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
    i.named ??= new Map();
    i.colors ??= new flux.Retrieve({
      definition: aetherColor.listDefinition,
      onChange: () => {
        i.named = new Map((i.colors.value ?? []).map((c) => [c.key, c]));
        i.requestRender("tool");
      },
      onError: (error) => i.runAsync(error, "Failed to retrieve colors"),
    });
    const client = synnax.use(ctx);
    if (client != null) i.client = client;
    i.requestRender("tool");
  }

  override afterDelete(): void {
    this.internal.retrieve.close();
    this.internal.colors.close();
    clearTimeout(this.internal.timer);
  }

  // Draws the strip again after delay milliseconds, or never when delay is undefined.
  // Each render sets it again, so the timer stops when the plot stops drawing.
  private redrawIn(delay: number | undefined): void {
    const { internal: i } = this;
    clearTimeout(i.timer);
    i.timer =
      delay == null
        ? undefined
        : setTimeout(() => i.requestRender("tool"), Math.min(delay, MAX_DELAY));
  }

  render(props: ProviderProps): void {
    const { dataToDecimalScale, region, viewport, timeRange, newest } = props;
    const { internal: i } = this;
    if (i.window == null || !covers(i.window, timeRange)) i.window = pad(timeRange);
    if (i.client != null) {
      i.retrieve.update(i.client, {
        where: { overlaps: i.window },
        limit: FETCH_LIMIT,
      });
      i.colors.update(i.client, {});
    }
    const { draw } = i;
    const now = TimeStamp.now();
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
      i.named,
      now,
      newest,
    );
    const bars = packed.filter(
      (b) => b.end >= box.left(region) && b.start <= box.right(region),
    );
    const count = packed.length;
    const height = stripHeight(lanes);
    const { cursor, visible, menuOpen, hovered: shown } = this.state;
    // A bar changes when its range starts. A running bar enters a view ahead of now
    // when now reaches the start of the view.
    const changes = visible
      ? packed.map((b) => (b.running ? timeRange.start : b.range.timeRange.start))
      : [];
    const delays = changes
      .filter((t) => now.before(t))
      .map((t) => new TimeRange(now, t).span.milliseconds);
    const running = visible ? bars.filter((b) => b.running) : [];
    if (running.some((b) => b.end <= box.right(region))) delays.push(HEAD_PERIOD);
    else if (running.length > 0) delays.push(CLOCK_PERIOD);
    this.redrawIn(delays.length === 0 ? undefined : Math.min(...delays));
    // The range of an open menu stays hovered while a live plot slides it away from the
    // cursor, so the menu keeps acting on the range the user clicked.
    let hovered: Bar | null = null;
    if (visible && menuOpen && shown != null)
      hovered = bars.find((b) => b.range.key === shown.key) ?? null;
    else if (visible && cursor != null) hovered = hitTest(bars, region, lanes, cursor);
    if (visible) {
      const clearScissor = draw.canvas.scissor(
        box.construct(
          { x: box.left(region) - 2, y: box.top(region) - height },
          { x: box.right(region) + 2, y: box.bottom(region) },
        ),
      );
      drawFills(draw, bars, region, hovered);
      drawBars(draw, bars, region, lanes, hovered, now);
      clearScissor();
      if (hovered != null && cursor != null && !menuOpen)
        drawTooltip(draw, hovered, cursor, region, now);
    }
    const viewportScale = dataToDecimalScale.scale(box.xBounds(viewport));
    let hoveredState: SelectedState | null = null;
    if (hovered != null) {
      const r = hovered.range;
      // A running range zooms to its newest sample, or to the axis end when not live.
      const end = hovered.running
        ? TimeStamp.min(r.timeRange.end, newest ?? timeRange.end)
        : r.timeRange.end;
      hoveredState = {
        key: r.key,
        parent: r.parent,
        name: r.name,
        color: r.color,
        labels: r.labels,
        timeRange: r.timeRange,
        hasChildren: r.hasChildren,
        viewport: {
          lower: viewportScale.pos(Number(r.timeRange.start.valueOf())),
          upper: viewportScale.pos(Number(end.valueOf())),
        },
      };
    }
    if (hoveredState != null) this.setState((s) => ({ ...s, hovered: hoveredState }));
    else if (shown != null && !menuOpen)
      this.setState((s) => ({ ...s, hovered: null }));
    if (this.state.count !== count || this.state.height !== height)
      this.setState((s) => ({ ...s, count, height }));
  }
}
