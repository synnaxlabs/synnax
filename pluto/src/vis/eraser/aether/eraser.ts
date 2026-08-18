// Copyright 2026 Synnax Labs, Inc.
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
  enabled: z.boolean().default(false),
});

interface InternalState {
  renderCtx: render.Context | null;
}

const CANVASES: render.CanvasVariant[] = ["gl", "lower2d", "upper2d"];

/**
 * Erases its region from the canvases on lifecycle changes. When no canvas render
 * context exists (a canvas-less mount), there is nothing to erase and Eraser no-ops.
 */
export class Eraser extends aether.Leaf<typeof eraserStateZ, InternalState> {
  static readonly TYPE = "eraser";
  schema = eraserStateZ;

  afterUpdate(ctx: aether.Context): void {
    if (this.deleted) return;
    this.internal.renderCtx = render.Context.useOptional(ctx);
    this.renderOnLifecycleChange();
  }

  afterDelete(): void {
    this.renderOnLifecycleChange();
  }

  renderOnLifecycleChange(): void {
    if (this.internal.renderCtx == null) return;
    this.internal.renderCtx.loop.set({
      key: `${this.type}-${this.key}`,
      render: this.render.bind(this),
      priority: "high",
      canvases: CANVASES,
    });
  }

  render(): void {
    if (this.deleted || !this.state.enabled) return;
    this.internal.renderCtx?.erase(this.state.region, xy.construct(0), ...CANVASES);
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Eraser.TYPE]: Eraser,
};
