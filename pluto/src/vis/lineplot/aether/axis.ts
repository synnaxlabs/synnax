// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  bounds,
  box,
  direction,
  scale,
  throttle,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { theming } from "@/theming/aether";
import { fontString } from "@/theming/core/fontString";
import { axis } from "@/vis/axis";
import { type TickType } from "@/vis/axis/ticks";
import { grid } from "@/vis/grid";
import { render } from "@/vis/render";

export const coreAxisStateZ = axis.axisStateZ
  .extend({
    bounds: bounds.bounds.optional(),
    autoBounds: z
      .object({
        lower: z.boolean().optional().default(true),
        upper: z.boolean().optional().default(true),
      })
      .or(z.boolean().optional().default(true)),
    autoBoundPadding: z.number().optional(),
    autoBoundUpdateInterval: TimeSpan.z.optional().default(TimeSpan.seconds(10)),
    size: z.number().optional().default(0),
    label: z.string().optional().default(""),
    labelSize: z.number().optional().default(0),
  })
  .partial({ color: true, font: true, gridColor: true });

export type BaseAxisState = z.output<typeof coreAxisStateZ>;

const AXIS_SIZE_UPDATE_UPPER_THRESHOLD = 2; // px;
const AXIS_SIZE_UPDATE_LOWER_THRESHOLD = 7; // px;

export const withinSizeThreshold = (prev: number, next: number): boolean =>
  bounds.contains(
    {
      lower: prev - AXIS_SIZE_UPDATE_LOWER_THRESHOLD,
      upper: prev + AXIS_SIZE_UPDATE_UPPER_THRESHOLD,
    },
    next,
  );

export const EMPTY_LINEAR_BOUNDS = bounds.DECIMAL;
const now = TimeStamp.now();
export const EMPTY_TIME_BOUNDS: bounds.Bounds = {
  lower: Number(now.valueOf()),
  upper: Number(now.add(TimeSpan.HOUR).valueOf()),
};

export const emptyBounds = (type: TickType): bounds.Bounds =>
  type === "linear" ? EMPTY_LINEAR_BOUNDS : EMPTY_TIME_BOUNDS;

export interface AxisRenderProps {
  grid: grid.Grid;
  plot: box.Box;
  viewport: box.Box;
  container: box.Box;
  canvases: render.CanvasVariant[];
  hold: boolean;
}

export const autoBounds = (
  b: bounds.Bounds[],
  padding: number = 0.1,
  type: TickType,
): bounds.Bounds => {
  const m = bounds.max(b.filter(bounds.isFinite));
  if (!bounds.isFinite(m)) return emptyBounds(type);
  const { lower, upper } = m;
  if (upper === lower) return { lower: lower - 1, upper: upper + 1 };
  const _padding = (upper - lower) * padding;
  return { lower: lower - _padding, upper: upper + _padding };
};

interface InternalState {
  render: render.Context;
  core: axis.Axis;
  // In the case where we're in a hold, we want to keep a snapshot of the hold bounds
  // so that we can rerender the plot in the same position even if the data changes.
  // changes.
  boundSnapshot?: bounds.Bounds;
  updateBounds?: (bounds: bounds.Bounds) => void;
}

const DEFAULT_X_BOUND_PADDING = 0.01;
const DEFAULT_Y_BOUND_PADDING = 0.1;

export class CoreAxis<
  S extends typeof coreAxisStateZ,
  C extends aether.Component = aether.Component,
> extends aether.Composite<S, InternalState, C> {
  async afterUpdate(ctx: aether.Context): Promise<void> {
    this.internal.render = render.Context.use(ctx);
    const theme = theming.use(ctx);
    this.state.autoBoundPadding ??=
      direction.construct(this.state.location) === "x"
        ? DEFAULT_Y_BOUND_PADDING
        : DEFAULT_X_BOUND_PADDING;
    this.internal.core = new axis.Canvas(this.internal.render, {
      color: theme.colors.gray.l8,
      font: fontString(theme, { level: "small" }),
      gridColor: theme.colors.gray.l1,
      ...this.state,
    });
    render.Controller.requestRender(ctx, render.REASON_LAYOUT);
    this.internal.updateBounds ??= throttle(
      (b) => this.setState((p) => ({ ...p, bounds: b })),
      this.state.autoBoundUpdateInterval.milliseconds,
    );
  }

  async afterDelete(ctx: aether.Context): Promise<void> {
    render.Controller.requestRender(ctx, render.REASON_LAYOUT);
  }

  renderAxis(props: AxisRenderProps, decimalToDataScale: scale.Scale): void {
    if (!props.canvases.includes("lower2d")) return;
    const { core } = this.internal;
    const { grid: g, container } = props;
    const position = grid.position(`${this.type}-${this.key}`, g, container);
    const p = {
      ...props,
      position,
      decimalToDataScale,
      size: this.state.size + this.state.labelSize,
    };
    const { size } = core.render(p);
    if (!withinSizeThreshold(this.state.size, size))
      this.setState((p) => ({ ...p, size }));
  }

  async bounds(
    hold: boolean,
    fetchDataBounds: () => Promise<bounds.Bounds[]>,
  ): Promise<[bounds.Bounds, Error | null]> {
    if (hold && this.internal.boundSnapshot != null)
      return [this.internal.boundSnapshot, null];
    const { lower, upper } = parseAutoBounds(this.state.autoBounds);
    if (!lower && !upper && this.state.bounds != null) {
      this.internal.boundSnapshot = this.state.bounds;
      return [this.state.bounds, null];
    }
    const merge = (auto: bounds.Bounds): bounds.Bounds => ({
      upper: upper || this.state.bounds == null ? auto.upper : this.state.bounds.upper,
      lower: lower || this.state.bounds == null ? auto.lower : this.state.bounds.lower,
    });
    let ab: bounds.Bounds;
    let err: Error | null = null;
    try {
      const dataBounds = await fetchDataBounds();
      ab = autoBounds(dataBounds, this.state.autoBoundPadding, this.state.type);
    } catch (err_) {
      ab = emptyBounds(this.state.type);
      err = err_ as Error;
    }
    const bounds = merge(ab);
    this.internal.boundSnapshot = bounds;
    if (
      this.state.bounds == null ||
      (lower && this.state.bounds.lower !== bounds.lower) ||
      (upper && this.state.bounds.upper !== bounds.upper)
    )
      this.internal.updateBounds?.(bounds);
    return [bounds, err];
  }

  async dataToDecimalScale(
    hold: boolean,
    fetchDataBounds: () => Promise<bounds.Bounds[]>,
    viewport: box.Box,
  ): Promise<[scale.Scale, Error | null]> {
    const [bounds, err] = await this.bounds(hold, fetchDataBounds);
    const dir = direction.swap(direction.construct(this.state.location));
    return [
      scale.Scale.scale<number>(bounds)
        .scale(1)
        .translate(-box.root(viewport)[dir])
        .magnify(1 / box.dim(viewport, dir)),
      err,
    ];
  }
}

export const parseAutoBounds = (
  autoBounds: boolean | { lower?: boolean; upper?: boolean },
): { lower: boolean; upper: boolean } => {
  if (typeof autoBounds === "boolean") return { lower: autoBounds, upper: autoBounds };
  return { lower: autoBounds?.lower ?? true, upper: autoBounds?.upper ?? true };
};
