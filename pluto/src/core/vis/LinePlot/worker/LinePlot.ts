// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box, XY } from "@synnaxlabs/x";
import { z } from "zod";

import { AetherComposite, Update } from "@/core/aether/worker";
import { XAxis } from "@/core/vis/LinePlot/worker/XAxis";
import { RenderController, RenderContext } from "@/core/vis/render";

export const linePlotState = z.object({
  plot: Box.z,
  container: Box.z,
  viewport: Box.z,
  clearOverscan: z.union([z.number(), XY.z]).optional().default(0),
});

export type LinePlotState = z.input<typeof linePlotState>;

export class LinePlot extends AetherComposite<typeof linePlotState, XAxis> {
  ctx: RenderContext;

  static readonly TYPE: string = "line-plot";

  constructor(update: Update) {
    super(update, linePlotState);
    this.ctx = RenderContext.use(update.ctx);
    RenderController.create(update.ctx, () => this.requestRender());
    this.onUpdate(() => this.handleUpdate());
  }

  private handleUpdate(): void {
    this.requestRender();
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

  private erase(): void {
    this.ctx.erase(this.region, this.clearOverScan);
  }

  private async render(): Promise<void> {
    this.erase();
    const removeScissor = this.ctx.scissorGL(this.plottingRegion);
    await Promise.all(
      this.children.map(
        async (xAxis) =>
          await xAxis.render({
            plottingRegion: this.plottingRegion,
            viewport: this.viewport,
          })
      )
    );
    removeScissor();
  }

  requestRender(): void {
    this.ctx.queue.push(this.key, async () => await this.render());
  }
}
