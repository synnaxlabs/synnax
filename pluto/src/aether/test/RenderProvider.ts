// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { aether } from "@/aether/aether";

const RENDER_CONTEXT_KEY = "pluto-render-context";

export const renderProviderStateZ = z.object({
  context: z.any(),
});

/** Thin Composite that injects a `render.Context`-shaped value into the parent context
 * under the key consumed by `render.Context.use`. Used by {@link mount} to bridge a
 * test recorder (or any duck-typed render context) into the aether tree without
 * requiring a real `Canvas` component (which depends on WebGL and OffscreenCanvas APIs
 * jsdom does not provide). */
export class RenderProvider extends aether.Composite<typeof renderProviderStateZ> {
  static readonly TYPE = "aetherTest.RenderProvider";
  static readonly stateZ = renderProviderStateZ;
  schema = RenderProvider.stateZ;

  afterUpdate(ctx: aether.Context): void {
    ctx.set(RENDER_CONTEXT_KEY, this.state.context);
  }
}
