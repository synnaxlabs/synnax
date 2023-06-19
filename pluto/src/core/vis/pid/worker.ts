// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box, XY } from "@synnaxlabs/x";
import { z } from "zod";

import { AetherComponent, AetherComposite, Update } from "@/core/aether/worker";
import { CSS } from "@/core/css";
import { RenderContext, RenderController } from "@/core/vis/render";

export const pidState = z.object({
  position: XY.z,
  region: Box.z,
});

export type PIDState = z.input<typeof pidState>;
export type ParsedPIDState = z.output<typeof pidState>;

interface PIDRenderProps {
  position: XY;
}

export interface PIDItem extends AetherComponent {
  render: (props: PIDRenderProps) => void;
}

export class PID extends AetherComposite<typeof pidState, PIDItem> {
  static readonly TYPE = CSS.B("pid");

  ctx: RenderContext;

  constructor(update: Update) {
    super(update, pidState);
    this.ctx = RenderContext.use(update.ctx);
    RenderController.control(update.ctx, () => this.requestRender());
    this.requestRender();
  }

  handleUpdate(): void {
    this.requestRender();
  }

  async render(): Promise<void> {
    this.ctx.eraseCanvas(new Box(this.state.region));
    this.children.forEach((child) => child.render({ position: this.state.position }));
  }

  private requestRender(): void {
    this.ctx.queue.push(this.key, async () => await this.render());
  }
}
