// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { aether } from "@/aether/aether";
import { color } from "@/color/core";
import { status } from "@/status/aether";
import { telem } from "@/telem/aether";

export const indicatorStateZ = z.object({
  statusSource: telem.statusSourceSpecZ.optional().default(telem.noopStatusSourceSpec),
  colorSource: telem.colorSourceSpecZ.optional().default(telem.noopColorSourceSpec),
  status: status.specZ,
  color: color.Color.z,
});

interface InternalState {
  statusSource: telem.StatusSource;
  colorSource: telem.ColorSource;
}

export class Indicator extends aether.Leaf<typeof indicatorStateZ, InternalState> {
  static readonly TYPE = "Indicator";
  schema = indicatorStateZ;

  afterUpdate(): void {
    const { internal: i } = this;
    i.statusSource = telem.useSource(this.ctx, this.state.statusSource, i.statusSource);
    i.colorSource = telem.useSource(this.ctx, this.state.colorSource, i.colorSource);
    void this.updateState();
    i.statusSource.onChange(() => {
      void this.updateState();
    });
    i.colorSource.onChange(() => {
      void this.updateState();
    });
  }

  afterDelete(): void {
    this.internal.statusSource.cleanup?.();
    this.internal.colorSource.cleanup?.();
  }

  async render(): Promise<void> {}

  async updateState(): Promise<void> {
    const color = await this.internal.colorSource.value();
    const status = await this.internal.statusSource.value();
    if (color.equals(this.state.color) && status.message === this.state.status.message)
      return;
    this.setState((p) => ({ ...p, color, status }));
  }
}
