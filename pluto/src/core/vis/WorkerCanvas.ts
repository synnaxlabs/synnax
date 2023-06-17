// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box } from "@synnaxlabs/x";
import { z } from "zod";

import { AetherComposite, AetherContext, Update } from "@/core/aether/worker";
import { LineGLProgramContext } from "@/core/vis/Line";
import { RenderContext } from "@/core/vis/render";

export const canvasState = z.object({
  dpr: z.number(),
  region: Box.z,
  glCanvas: z.instanceof(OffscreenCanvas).optional(),
  canvasCanvas: z.instanceof(OffscreenCanvas).optional(),
});

export type CanvasState = z.input<typeof canvasState>;
export type ParsedCanvasState = z.output<typeof canvasState>;

export class Canvas extends AetherComposite<typeof canvasState> {
  static readonly TYPE = "canvas";

  constructor(update: Update) {
    super(update, canvasState);
    this.onUpdate((ctx: AetherContext) => {
      let renderCtx = RenderContext.useOptional(ctx);
      if (renderCtx == null) {
        const { glCanvas, canvasCanvas } = this.state;
        if (glCanvas == null || canvasCanvas == null) throw new Error("unexpected");
        renderCtx = RenderContext.create(ctx, glCanvas, canvasCanvas);
        LineGLProgramContext.create(ctx);
      }
      renderCtx.resize(new Box(this.state.region), this.state.dpr);
    });
  }
}

export const bootstrap = canvasState.extend({
  key: z.string(),
  glCanvas: z.instanceof(OffscreenCanvas),
  canvasCanvas: z.instanceof(OffscreenCanvas),
});

export type Bootstrap = z.output<typeof bootstrap>;
