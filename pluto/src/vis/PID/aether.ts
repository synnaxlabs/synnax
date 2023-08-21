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

const pidState = z.object({
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

export class AetherPID extends aether.Composite<
  typeof pidState,
  InternalState,
  PIDElement
> {
  static readonly TYPE = CSS.B("pid");
  static readonly stateZ = pidState;
  schema = AetherPID.stateZ;

  afterUpdate(): void {
    this.internal.render = render.Context.use(this.ctx);
    render.Controller.control(this.ctx, () => this.requestRender());
    this.requestRender();
    if (this.state.error != null) this.setState((p) => ({ ...p, error: undefined }));
  }

  afterDelete(): void {
    this.requestRender();
  }

  get region(): Box {
    return new Box(this.state.region);
  }

  get prevRegion(): Box {
    return new Box(this.prevState.region);
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

    return async () => {
      renderCtx.eraseCanvas(this.prevRegion.isZero ? this.region : this.prevRegion);
    };
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
