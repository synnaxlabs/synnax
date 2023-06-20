// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError } from "@synnaxlabs/client";
import { Box } from "@synnaxlabs/x";
import { z } from "zod";

import { AetherComposite, AetherContext, Update } from "@/core/aether/worker";
import { LineGLProgramContext } from "@/core/vis/Line/LineGL";
import { RenderContext } from "@/core/vis/render";

export const canvasState = z.object({
  dpr: z.number(),
  region: Box.z,
  glCanvas: z.instanceof(OffscreenCanvas).optional(),
  upper2dCanvas: z.instanceof(OffscreenCanvas).optional(),
  lower2dCanvas: z.instanceof(OffscreenCanvas).optional(),
});

export type CanvasState = z.input<typeof canvasState>;
export type ParsedCanvasState = z.output<typeof canvasState>;

export class Canvas extends AetherComposite<typeof canvasState> {
  static readonly TYPE = "canvas";

  constructor(update: Update) {
    super(update, canvasState);
  }

  handleUpdate(ctx: AetherContext): void {
    let renderCtx = RenderContext.useOptional(ctx);
    if (renderCtx == null) {
      const { glCanvas, lower2dCanvas, upper2dCanvas } = this.state;
      if (glCanvas == null || lower2dCanvas == null || upper2dCanvas == null)
        throw new UnexpectedError(
          "[vis.worker.Canvas] - expected render context bootstrap to include all canvases"
        );
      renderCtx = RenderContext.create(ctx, glCanvas, lower2dCanvas, upper2dCanvas);
      LineGLProgramContext.create(ctx);
    } else {
      renderCtx.update(ctx);
    }
    renderCtx.resize(new Box(this.state.region), this.state.dpr);
  }
}
