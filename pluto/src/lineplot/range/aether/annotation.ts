// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { aether } from "@synnaxlabs/lyra/aether/runtime";
import { theming } from "@synnaxlabs/lyra/theming/aether";
import { telem } from "@synnaxlabs/x/telem";
import { z } from "zod";

import { Draw2D } from "@/vis/draw2d";
import { render } from "@/vis/render";

export const annotationStateZ = z.object({ start: telem.TimeStamp.z, end: telem.TimeStamp.z });

interface InternalState {
  render: render.Context;
  draw: Draw2D;
}

export class Annotation extends aether.Leaf<typeof annotationStateZ, InternalState> {
  static readonly TYPE = "range-annotation";
  schema = annotationStateZ;

  afterUpdate(ctx: aether.Context): void {
    this.internal.render = render.Context.use(ctx);
    this.internal.draw = new Draw2D(this.internal.render.upper2d, theming.use(ctx));
  }
}
