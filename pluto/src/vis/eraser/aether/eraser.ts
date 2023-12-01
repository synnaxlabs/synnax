// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { render } from "@/vis/render";

export const eraserStateZ = z.object({
  region: box.box,
});

interface InternalState {
  render: render.Context;
}

export class Eraser extends aether.Leaf<typeof eraserStateZ, InternalState> {
  static readonly TYPE = "eraser";
  schema = eraserStateZ;

  private readonly eraser: render.Eraser = new render.Eraser();

  afterUpdate(): void {
    this.internal.render = render.Context.use(this.ctx);
    this.internal.render.queue.set({
      key: `${this.type}-${this.key}`,
      render: this.render.bind(this),
      priority: "high",
      canvases: ["gl", "lower2d", "upper2d"],
    });
  }

  async render(): Promise<render.Cleanup> {
    return async ({ canvases }) => {
      this.eraser.erase(
        this.internal.render,
        this.state.region,
        this.prevState.region,
        xy.construct(0),
        canvases,
      );
    };
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Eraser.TYPE]: Eraser,
};
