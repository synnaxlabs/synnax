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

import { AetherComponent, AetherComposite, Update } from "@/core/aether/worker";
import { CSS } from "@/core/css";
import { RenderContext, RenderController } from "@/core/vis/render";

const pidState = z.object({
  position: XY.z,
  region: Box.z,
  error: z.string().optional(),
});

interface PIDElementProps {
  position: XY;
}

export interface PIDElement extends AetherComponent {
  render: (props: PIDElementProps) => Promise<void>;
}

export class AetherPID extends AetherComposite<typeof pidState, PIDElement> {
  static readonly TYPE = CSS.B("pid");
  static readonly stateZ = pidState;

  renderCtx: RenderContext;

  constructor(update: Update) {
    super(update, pidState);
    this.renderCtx = RenderContext.use(update.ctx);
    RenderController.control(update.ctx, () => this.requestRender());
    this.requestRender();
  }

  handleUpdate(): void {
    this.requestRender();
  }

  handleDelete(): void {
    this.renderCtx.erase(new Box(this.prevState.region));
  }

  async render(): Promise<void> {
    const region = new Box(this.state.region);
    const prevRegion = new Box(this.prevState.region);
    this.renderCtx.eraseCanvas(prevRegion.isZero ? region : prevRegion);
    const clearScissor = this.renderCtx.scissorCanvas(region);
    try {
      await Promise.all(
        this.children.map(
          async (child) =>
            await child.render({
              position: region.topLeft.translate(this.state.position),
            })
        )
      );
    } catch (e) {
      this.setState((p) => ({ ...p, error: (e as Error).message }));
    } finally {
      clearScissor();
    }
  }

  private requestRender(): void {
    this.renderCtx.queue.push(this.key, this.render.bind(this));
  }
}
