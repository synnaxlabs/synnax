// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { scale, xy, box } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { CSS } from "@/css";
import { status } from "@/status/aether";
import { render } from "@/vis/render";

export const pidStateZ = z.object({
  position: xy.xy,
  zoom: z.number(),
  region: box.box,
});

interface PIDElementProps {
  s?: scale.XY;
}

export interface PIDElement extends aether.Component {
  render: (props: PIDElementProps) => Promise<void>;
}

interface InternalState {
  render: render.Context;
  aggregate: status.Aggregate;
}

const CANVASES: render.CanvasVariant[] = ["upper2d", "lower2d"];

export class PID extends aether.Composite<typeof pidStateZ, InternalState, PIDElement> {
  static readonly TYPE = CSS.B("pid");
  static readonly stateZ = pidStateZ;
  readonly eraser: render.Eraser = new render.Eraser();
  schema = PID.stateZ;

  afterUpdate(): void {
    this.internal.render = render.Context.use(this.ctx);
    this.internal.aggregate = status.useAggregate(this.ctx);
    render.Controller.control(this.ctx, () => this.requestRender());
    this.requestRender();
  }

  afterDelete(): void {
    this.requestRender();
  }

  async render(): Promise<render.Cleanup> {
    if (this.deleted) return async () => {};
    const { render: renderCtx } = this.internal;
    const region = box.construct(this.state.region);
    const clearScissor = renderCtx.scissor(region, xy.ZERO, CANVASES);
    try {
      await Promise.all(
        this.children.map(
          async (child) =>
            await child.render({
              s: scale.XY.magnify(xy.construct(this.state.zoom))
                .translate(box.topLeft(region))
                .translate(this.state.position),
            }),
        ),
      );
    } catch (e) {
      this.internal.aggregate({
        variant: "error",
        message: (e as Error).message,
      });
    } finally {
      clearScissor();
    }

    return async () =>
      this.eraser.erase(
        this.internal.render,
        this.state.region,
        this.prevState.region,
        xy.construct(10),
        CANVASES,
      );
  }

  private requestRender(): void {
    const { render: renderCtx } = this.internal;
    renderCtx.queue.push({
      key: this.key,
      render: this.render.bind(this),
      priority: "high",
      canvases: CANVASES,
    });
  }
}

export const REGISTRY = {
  [PID.TYPE]: PID,
};
