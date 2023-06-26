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
import { CSS } from "@/core/css";
import { XAxis } from "@/core/vis/LinePlot/aether/XAxis";
import { RenderController, RenderContext } from "@/core/vis/render";

const linePlotState = z.object({
  plot: Box.z,
  container: Box.z,
  viewport: Box.z,
  clearOverscan: z.union([z.number(), XY.z]).optional().default(10),
  error: z.string().optional(),
});

export class LinePlot extends AetherComposite<typeof linePlotState, XAxis> {
  renderCtx: RenderContext;

  static readonly TYPE: string = CSS.B("line-plot");
  static readonly stateZ = linePlotState;

  constructor(update: Update) {
    super(update, linePlotState);
    this.renderCtx = RenderContext.use(update.ctx);
    RenderController.control(update.ctx, () => this.requestRender());
  }

  handleUpdate(): void {
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
    this.renderCtx.erase(this.region, this.clearOverScan);
  }

  private async render(): Promise<void> {
    try {
      this.erase();
      const removeScissor = this.renderCtx.scissorGL(this.plottingRegion);
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
    } catch (e) {
      this.setState((p) => ({ ...p, error: (e as Error).message }));
    }
  }

  requestRender(): void {
    this.renderCtx.queue.push(this.key, async () => await this.render());
  }
}
