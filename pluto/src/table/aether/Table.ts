// Copyright 2024 Synnax Labs, Inc.
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
import { render } from "@/vis/render";

export const tableStateZ = z.object({
  region: box.box,
  clearOverScan: xy.crudeZ.optional().default(0),
  visible: z.boolean().optional().default(true),
});

interface CellProps {
  viewportScale: scale.XY;
}

export interface Cell extends aether.Component {
  render: ({ viewportScale }: CellProps) => Promise<void>;
}

interface InternalState {
  renderCtx: render.Context;
}

const CANVASES: render.CanvasVariant[] = ["upper2d", "lower2d"];

export class Table extends aether.Composite<typeof tableStateZ, InternalState, Cell> {
  static readonly TYPE = "Table";
  static readonly stateZ = tableStateZ;
  schema = Table.stateZ;

  async afterUpdate(): Promise<void> {
    const { internal: i } = this;
    i.renderCtx = render.Context.use(this.ctx);
    render.Controller.control(this.ctx, () => this.requestRender("low"));
    void this.render();
  }

  async render(): Promise<render.Cleanup | undefined> {
    if (!this.state.visible) return;
    const { renderCtx } = this.internal;
    const viewportScale = scale.XY.translate(box.topLeft(this.state.region));
    const clearScissor = renderCtx.scissor(
      this.state.region,
      xy.construct(this.state.clearOverScan),
      CANVASES,
    );

    try {
      await Promise.all(this.children.map((child) => child.render({ viewportScale })));
    } finally {
      clearScissor();
    }
    const eraseRegion = box.copy(this.state.region);
    return async () => {
      this.internal.renderCtx.erase(eraseRegion, this.state.clearOverScan);
    };
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
