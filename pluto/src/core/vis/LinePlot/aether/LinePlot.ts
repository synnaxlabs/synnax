// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box, Compare, CrudeOuterLocation, Location, XY, order } from "@synnaxlabs/x";
import { z } from "zod";

import { LookupResult } from "../../Line/core";

import { AetherComposite } from "@/core/aether/worker";
import { CSS } from "@/core/css";
import { AetherXAxis } from "@/core/vis/LinePlot/aether/XAxis";
import {
  RenderController,
  RenderContext,
  RenderCleanup,
  RenderPriority,
} from "@/core/vis/render";
const gridPositionMeta = z.object({
  key: z.string(),
  size: z.number(),
  order,
  loc: Location.strictOuterZ,
});

export type GridPositionMeta = z.input<typeof gridPositionMeta>;

export const filterAxisLoc = (
  loc: CrudeOuterLocation,
  grid: GridPositionMeta[]
): GridPositionMeta[] =>
  grid
    .filter(({ loc: l }) => new Location(l).equals(loc))
    .sort((a, b) => Compare.order(a.order, b.order));

const linePlotState = z.object({
  plot: Box.z,
  container: Box.z,
  viewport: Box.z,
  clearOverscan: z.union([z.number(), XY.z]).optional().default(10),
  error: z.string().optional(),
  grid: z.array(gridPositionMeta),
});
interface Derived {
  ctx: RenderContext;
}

export class AetherLinePlot extends AetherComposite<
  typeof linePlotState,
  Derived,
  AetherXAxis
> {
  static readonly TYPE: string = CSS.B("LinePlot");

  static readonly z = linePlotState;
  schema = AetherLinePlot.z;

  derive(): Derived {
    return { ctx: RenderContext.use(this.ctx) };
  }

  afterUpdate(): void {
    RenderController.control(this.ctx, () => this.requestRender("low"));
    this.requestRender("high");
  }

  afterDelete(): void {
    this.requestRender("high");
  }

  private get plottingRegion(): Box {
    return new Box(this.state.plot);
  }

  private get region(): Box {
    return new Box(this.state.container);
  }

  private get viewport(): Box {
    return new Box(this.state.viewport);
  }

  private get clearOverScan(): XY {
    return new XY(
      typeof this.state.clearOverscan === "number"
        ? { x: this.state.clearOverscan, y: this.state.clearOverscan }
        : this.state.clearOverscan
    );
  }

  async lookupX(x: number): Promise<LookupResult[]> {
    return (
      await Promise.all(
        this.childrenOfType(AetherXAxis.TYPE).flatMap(
          async (xAxis) =>
            await xAxis.lookupX(
              {
                plottingRegion: this.plottingRegion,
                viewport: this.viewport,
                region: this.region,
                grid: this.state.grid,
              },
              x
            )
        )
      )
    ).flat();
  }

  private async render(): Promise<RenderCleanup> {
    if (this.deleted) return async () => {};
    const { ctx } = this.derived;
    const removeGlScissor = ctx.scissorGL(this.plottingRegion);
    const removeCanvasScissor = ctx.scissorCanvas(this.region);

    try {
      await Promise.all(
        this.childrenOfType(AetherXAxis.TYPE).map(
          async (xAxis, i) =>
            await xAxis.render({
              plottingRegion: this.plottingRegion,
              viewport: this.viewport,
              region: this.region,
              grid: this.state.grid,
            })
        )
      );
    } catch (e) {
      this.setState((p) => ({ ...p, error: (e as Error).message }));
      throw e;
    } finally {
      removeGlScissor();
      removeCanvasScissor();
    }
    return async () => ctx.erase(new Box(this.prevState.container), this.clearOverScan);
  }

  requestRender(priority: RenderPriority): void {
    const { ctx } = this.derived;
    ctx.queue.push({
      key: `${this.type}-${this.key}`,
      render: this.render.bind(this),
      priority,
    });
  }
}

export const calculateAxisPosition = (
  key: string,
  grid: GridPositionMeta[],
  plottingRegion: Box
): XY => {
  const axis = grid.find(({ key: k }) => k === key);
  if (axis == null) return XY.ZERO;
  const loc = new Location(axis.loc);
  const axes = filterAxisLoc(loc.crude as CrudeOuterLocation, grid);
  const index = axes.findIndex(({ key: k }) => k === key);
  const offset = axes.slice(0, index).reduce((acc, { size }) => acc + size, 0);
  switch (loc.crude) {
    case "left":
      return plottingRegion.topLeft.translateX(-offset - axis.size);
    case "right":
      return plottingRegion.topRight.translateX(offset);
    case "top":
      return plottingRegion.topLeft.translateY(-offset - axis.size);
    default:
      return plottingRegion.bottomLeft.translateY(offset);
  }
};
