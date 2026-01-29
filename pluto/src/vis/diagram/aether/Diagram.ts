// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, color, primitive, scale, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { status } from "@/status/aether";
import { render } from "@/vis/render";

export const diagramStateZ = z.object({
  position: xy.xyZ,
  zoom: z.number(),
  region: box.box,
  clearOverScan: xy.crudeZ.default(5),
  visible: z.boolean().default(true),
  autoRenderInterval: z.number().optional(),
});

interface ElementProps {
  viewportScale?: scale.XY;
}

export interface Element extends aether.Component {
  render?: (props: ElementProps) => void;
}

interface InternalState {
  renderCtx: render.Context;
  viewportScale: scale.XY;
  handleError: status.ErrorHandler;
  autoRenderInterval: ReturnType<typeof setInterval>;
}

const CANVASES: render.CanvasVariant[] = ["upper2d", "lower2d"];

export class Diagram extends aether.Composite<
  typeof diagramStateZ,
  InternalState,
  Element
> {
  static readonly TYPE = "Diagram";
  static readonly stateZ = diagramStateZ;
  schema = Diagram.stateZ;

  afterUpdate(ctx: aether.Context): void {
    this.internal.renderCtx = render.Context.use(ctx);
    this.internal.handleError = status.useErrorHandler(ctx);
    if (primitive.isNonZero(this.state.autoRenderInterval))
      this.internal.autoRenderInterval ??= setInterval(() => {
        if (this.state.visible) this.requestRender("low");
      }, this.state.autoRenderInterval);

    render.control(ctx, () => {
      if (!this.state.visible) return;
      this.requestRender("low");
    });
    if (!this.state.visible && !this.prevState.visible) return;
    this.internal.viewportScale = scale.XY.magnify(xy.construct(this.state.zoom))
      .translate(box.topLeft(this.state.region))
      .translate(this.state.position);
    this.requestRender("high");
  }

  afterDelete(): void {
    if (this.internal.autoRenderInterval != null)
      clearInterval(this.internal.autoRenderInterval);
    this.requestRender("high");
  }

  render(): render.Cleanup | undefined {
    if (this.deleted) return undefined;
    const { renderCtx, handleError, viewportScale } = this.internal;
    const region = box.construct(this.state.region);
    if (!this.state.visible)
      return () => renderCtx.erase(region, this.state.clearOverScan, ...CANVASES);
    const clearScissor = renderCtx.scissor(region, xy.ZERO, CANVASES);
    try {
      this.children.forEach((child) => child.render?.({ viewportScale }));
    } catch (e) {
      handleError(e, "failed to render diagram");
    } finally {
      clearScissor();
    }
    const eraseRegion = box.copy(this.state.region);
    return () => {
      renderCtx.lower2d.fillStyle = color.hex(color.BLACK);
      renderCtx.lower2d.fillRect(
        ...xy.couple(box.topLeft(eraseRegion)),
        box.width(eraseRegion),
        box.height(eraseRegion),
      );
      renderCtx.erase(eraseRegion, this.state.clearOverScan, ...CANVASES);
    };
  }

  private requestRender(priority: render.Priority): void {
    const { renderCtx } = this.internal;
    renderCtx.loop.set({
      key: `${Diagram.TYPE}-${this.key}`,
      render: this.render.bind(this),
      priority,
      canvases: CANVASES,
    });
  }
}

export const REGISTRY = {
  [Diagram.TYPE]: Diagram,
};
