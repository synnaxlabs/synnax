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
import { telem } from "@/telem/core";
import { noop } from "@/telem/noop";

export const indicatorStateZ = z.object({
  statusSource: telem.statusSourceSpecZ.optional().default(noop.statusSourceSpec),
  colorSource: telem.colorSourceSpecZ.optional().default(noop.colorSourceSpec),
  status: status.specZ,
  color: color.Color.z,
});

interface InternalState {
  statusSource: telem.StatusSource;
  cleanupStatusSource: () => void;
  colorSource: telem.ColorSource;
  cleanupColorSource: () => void;
}

export class Indicator extends aether.Leaf<typeof indicatorStateZ, InternalState> {
  static readonly TYPE = "Indicator";
  schema = indicatorStateZ;

  afterUpdate(): void {
    const [statusSource, cleanupStatusSource] = telem.use<telem.StatusSource>(
      this.ctx,
      `${this.key}-statusSource`,
      this.state.statusSource,
    );
    this.internal.statusSource = statusSource;
    this.internal.cleanupStatusSource = cleanupStatusSource;

    const [colorSource, cleanupColorSource] = telem.use<telem.ColorSource>(
      this.ctx,
      `${this.key}-colorSource`,
      this.state.colorSource,
    );
    this.internal.colorSource = colorSource;
    this.internal.cleanupColorSource = cleanupColorSource;

    void this.forwardChange();

    this.internal.statusSource.onChange(() => {
      void this.forwardChange();
    });
    this.internal.colorSource.onChange(() => {
      void this.forwardChange();
    });
  }

  afterDelete(): void {
    this.internal.cleanupStatusSource();
    this.internal.cleanupColorSource();
  }

  async render(): Promise<void> {}

  async forwardChange(): Promise<void> {
    const color = await this.internal.colorSource.color();
    const status = await this.internal.statusSource.status();
    if (color.equals(this.state.color) && status.message === this.state.status.message)
      return;
    this.setState((p) => ({ ...p, color, status }));
  }
}
