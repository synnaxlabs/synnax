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

const pidState = z.object({
  position: XY.z,
  region: Box.z,
});

interface PIDRenderProps {
  position: XY;
}

export interface PIDItem extends AetherComponent {
  render: (props: PIDRenderProps) => void;
}

export class AetherPID extends AetherComposite<typeof pidState, PIDItem> {
  static readonly TYPE = CSS.B("pid");
  static readonly stateZ = pidState;

  renderCtx: RenderContext;

  constructor(update: Update) {
    super(update, pidState);
    this.renderCtx = RenderContext.use(update.ctx);
    RenderController.control(update.ctx, () => this.requestRender());
    this.requestRender();
  }

  handleUpdate(): void {
    this.requestRender();
  }

  async render(): Promise<void> {
    this.renderCtx.eraseCanvas(new Box(this.state.region));
    this.children.forEach((child) => child.render({ position: this.state.position }));
  }

  private requestRender(): void {
    this.renderCtx.queue.push(this.key, this.render.bind(this));
  }
}
