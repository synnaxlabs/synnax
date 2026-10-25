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
import { Context } from "@/vis/line/aether/line";
import { render } from "@/vis/render";

/**
 * Supplies the line GL context and a render requestor, as the canvas and the plot do in
 * production. A plot mounted under it supplies its own requestor.
 */
export class Host extends aether.Composite<typeof Host.stateZ> {
  static readonly TYPE = "line-test-host";
  static readonly stateZ = z.object({});
  schema = Host.stateZ;

  afterUpdate(ctx: aether.Context): void {
    Context.create(ctx, render.Context.use(ctx));
    render.control(ctx, () => {});
  }
}
