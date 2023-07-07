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

import {
  AetherComponentRegistry,
  AetherComposite,
  AetherContext,
} from "@/core/aether/worker";
import { LineGLProgramContext } from "@/core/vis/Line/LineGL";
import { RenderContext } from "@/core/vis/render";

const canvasState = z.object({
  dpr: z.number(),
  region: Box.z,
  bootstrap: z.boolean().optional().default(false),
  glCanvas: z.instanceof(OffscreenCanvas).optional(),
  upper2dCanvas: z.instanceof(OffscreenCanvas).optional(),
  lower2dCanvas: z.instanceof(OffscreenCanvas).optional(),
});

export class AetherCanvas extends AetherComposite<typeof canvasState> {
  static readonly TYPE = "Canvas";
  static readonly z = canvasState;
  static readonly REGISTRY: AetherComponentRegistry = {
    [AetherCanvas.TYPE]: (u) => new AetherCanvas(u),
  };

  schema = canvasState;

  derive(): void {
    let renderCtx = RenderContext.useOptional(this.ctx);
    if (renderCtx == null) {
      if (!this.state.bootstrap) return;
      const { glCanvas, lower2dCanvas, upper2dCanvas } = this.state;
      if (glCanvas == null || lower2dCanvas == null || upper2dCanvas == null)
        throw new UnexpectedError(
          "[vis.worker.Canvas] - expected render context bootstrap to include all canvases"
        );
      renderCtx = RenderContext.create(
        this.ctx,
        glCanvas,
        lower2dCanvas,
        upper2dCanvas
      );
      LineGLProgramContext.create(this.ctx);
    } else renderCtx.update(this.ctx);
    renderCtx.resize(this.state.region, this.state.dpr);
  }
}
