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
    if (this.deleted) return;
    this.internal.render = render.Context.use(this.ctx);
    void this.internal.render.loop.set({
      key: `${this.type}-${this.key}`,
      render: this.render.bind(this),
      priority: "high",
      canvases: ["gl", "lower2d", "upper2d"],
    });
  }

  afterDelete(): void {
    void this.internal.render.loop.set({
      key: `${this.type}-${this.key}`,
      render: this.render.bind(this),
      priority: "high",
      canvases: ["gl", "lower2d", "upper2d"],
    });
  }

  async render(): Promise<undefined> {
    if (this.deleted) return;
    this.eraser.erase(
      this.internal.render,
      this.state.region,
      this.prevState.region,
      xy.construct(0),
      ["gl", "lower2d", "upper2d"],
    );
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Eraser.TYPE]: Eraser,
};
