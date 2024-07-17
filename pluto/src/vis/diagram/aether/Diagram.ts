// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, scale, xy } from "@synnaxlabs/x";
import { useRef } from "react";
import { useReactFlow } from "reactflow";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { status } from "@/status/aether";
import { translateViewportBackward, type Viewport } from "@/vis/diagram/aether/types";
import { render } from "@/vis/render";

export const diagramStateZ = z.object({
  position: xy.xy,
  zoom: z.number(),
  region: box.box,
  clearOverScan: xy.crudeZ.optional().default(10),
});

interface ElementProps {
  viewportScale?: scale.XY;
}

export interface Element extends aether.Component {
  render: (props: ElementProps) => Promise<void>;
}

interface InternalState {
  renderCtx: render.Context;
  addStatus: status.Aggregate;
}

const CANVASES: render.CanvasVariant[] = ["upper2d", "lower2d"];

export const useInitialViewport = (): Viewport => {
  const flow = useReactFlow();
  return useRef<Viewport>(translateViewportBackward(flow.getViewport())).current;
};

export class Diagram extends aether.Composite<
  typeof diagramStateZ,
  InternalState,
  Element
> {
  static readonly TYPE = "Diagram";
  static readonly stateZ = diagramStateZ;
  readonly eraser: render.Eraser = new render.Eraser();
  schema = Diagram.stateZ;

  async afterUpdate(): Promise<void> {
    this.internal.renderCtx = render.Context.use(this.ctx);
    this.internal.addStatus = status.useAggregate(this.ctx);
    render.Controller.control(this.ctx, () => this.requestRender("low"));
    this.requestRender("high");
  }

  async afterDelete(): Promise<void> {
    this.requestRender("high");
  }

  async render(): Promise<render.Cleanup | undefined> {
    if (this.deleted) return undefined;
    const { renderCtx, addStatus } = this.internal;
    const { zoom, position } = this.state;
    const region = box.construct(this.state.region);
    const clearScissor = renderCtx.scissor(region, xy.ZERO, CANVASES);
    const viewportScale = scale.XY.magnify(xy.construct(zoom))
      .translate(box.topLeft(region))
      .translate(position);

    try {
      await Promise.all(
        this.children.map(async (child) => await child.render({ viewportScale })),
      );
    } catch (e) {
      addStatus({ variant: "error", message: (e as Error).message });
    } finally {
      clearScissor();
    }

    const eraseRegion = box.copy(this.state.region);
    return async () => {
      this.internal.renderCtx.erase(eraseRegion, this.state.clearOverScan, ...CANVASES);
    };
  }

  private requestRender(priority: render.Priority): void {
    const { renderCtx } = this.internal;
    void renderCtx.loop.set({
      key: `${Diagram.TYPE}-${this.key}`,
      render: async () => await this.render(),
      priority,
      canvases: CANVASES,
    });
  }
}

export const REGISTRY = {
  [Diagram.TYPE]: Diagram,
};
