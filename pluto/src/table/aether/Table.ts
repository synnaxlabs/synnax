// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, scale, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { status } from "@/status/aether";
import { render } from "@/vis/render";

export const tableStateZ = z.object({
  region: box.box,
  clearOverScan: xy.crudeZ.default(0),
  visible: z.boolean().default(true),
  autoRenderInterval: z.number().default(1000),
});

interface CellProps {
  viewportScale: scale.XY;
}

export interface Cell extends aether.Component {
  render: ({ viewportScale }: CellProps) => void;
}

interface InternalState {
  renderCtx: render.Context;
  handleError: status.ErrorHandler;
  autoRenderInterval: ReturnType<typeof setInterval>;
}

const CANVASES: render.CanvasVariant[] = ["upper2d", "lower2d"];

export class Table extends aether.Composite<typeof tableStateZ, InternalState, Cell> {
  static readonly TYPE = "Table";
  static readonly stateZ = tableStateZ;
  schema = Table.stateZ;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.renderCtx = render.Context.use(ctx);
    i.handleError = status.useErrorHandler(ctx);
    i.autoRenderInterval ??= setInterval(
      () => this.state.visible && this.requestRender("low"),
      this.state.autoRenderInterval,
    );
    render.control(ctx, () => {
      if (!this.state.visible) return;
      this.requestRender("low");
    });
    if (!this.state.visible && !this.prevState.visible) return;
    this.requestRender("high");
  }

  afterDelete(): void {
    if (this.internal.autoRenderInterval != null)
      clearInterval(this.internal.autoRenderInterval);
    this.requestRender("high");
  }

  render(): render.Cleanup | undefined {
    if (this.deleted) return;
    const eraseRegion = box.copy(this.state.region);
    if (!this.state.visible)
      return () => this.internal.renderCtx.erase(eraseRegion, this.state.clearOverScan);
    const { renderCtx, handleError } = this.internal;
    const viewportScale = scale.XY.translate(box.topLeft(this.state.region));
    const clearScissor = renderCtx.scissor(
      this.state.region,
      xy.construct(this.state.clearOverScan),
      CANVASES,
    );

    try {
      for (const child of this.children) child.render({ viewportScale });
    } catch (e) {
      handleError(e, "Failed to render table");
    } finally {
      clearScissor();
    }
    return () => this.internal.renderCtx.erase(eraseRegion, this.state.clearOverScan);
  }

  private requestRender(priority: render.Priority): void {
    const { renderCtx } = this.internal;
    void renderCtx.loop.set({
      key: `${Table.TYPE}-${this.key}`,
      render: this.render.bind(this),
      priority,
      canvases: CANVASES,
    });
  }
}

export const REGISTRY = {
  [Table.TYPE]: Table,
};
