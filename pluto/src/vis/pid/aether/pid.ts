// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box, XY, XYScale } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { CSS } from "@/css";
import { render } from "@/vis/render";

export const pidStateZ = z.object({
  position: XY.z,
  zoom: z.number(),
  region: Box.z,
  error: z.string().optional(),
});

interface PIDElementProps {
  scale?: XYScale;
}

export interface PIDElement extends aether.Component {
  render: (props: PIDElementProps) => Promise<void>;
}

interface InternalState {
  render: render.Context;
}

export class PID extends aether.Composite<typeof pidStateZ, InternalState, PIDElement> {
  static readonly TYPE = CSS.B("pid");
  static readonly stateZ = pidStateZ;
  readonly eraser: render.Eraser = new render.Eraser();
  schema = PID.stateZ;

  afterUpdate(): void {
    this.internal.render = render.Context.use(this.ctx);
    render.Controller.control(this.ctx, () => this.requestRender());
    this.requestRender();
    if (this.state.error != null) this.setState((p) => ({ ...p, error: undefined }));
  }

  afterDelete(): void {
    this.requestRender();
  }

  async render(): Promise<render.Cleanup> {
    if (this.deleted) return async () => {};
    const { render: renderCtx } = this.internal;
    const region = new Box(this.state.region);
    const clearScissor = renderCtx.scissorCanvas(region);
    try {
      await Promise.all(
        this.children.map(
          async (child) =>
            await child.render({
              scale: XYScale.magnify(new XY(this.state.zoom))
                .translate(region.topLeft)
                .translate(this.state.position),
            })
        )
      );
    } catch (e) {
      this.setState((p) => ({ ...p, error: (e as Error).message }));
    } finally {
      clearScissor();
    }

    return async () =>
      this.eraser.erase(
        this.internal.render,
        this.state.region,
        this.prevState.region,
        new XY(10)
      );
  }

  private requestRender(): void {
    const { render: renderCtx } = this.internal;
    renderCtx.queue.push({
      key: this.key,
      render: this.render.bind(this),
      priority: "high",
    });
  }
}

export const REGISTRY = {
  [PID.TYPE]: PID,
};
