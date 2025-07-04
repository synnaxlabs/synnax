// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { z } from "zod/v4";

import { aether } from "@/aether/aether";
import { status } from "@/status/aether";
import { telem } from "@/telem/aether";

export const indicatorStateZ = z.object({
  statusSource: telem.statusSourceSpecZ.optional().default(telem.noopStatusSourceSpec),
  colorSource: telem.colorSourceSpecZ.optional().default(telem.noopColorSourceSpec),
  status: status.specZ,
  color: color.colorZ.optional(),
});

interface InternalState {
  statusSource: telem.StatusSource;
  colorSource: telem.ColorSource;
}

export class Indicator extends aether.Leaf<typeof indicatorStateZ, InternalState> {
  static readonly TYPE = "Indicator";
  schema = indicatorStateZ;
  stopListeningStatus?: () => void;
  stopListeningColor?: () => void;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    const { statusSource, colorSource } = this.state;
    i.statusSource = telem.useSource(ctx, statusSource, i.statusSource);
    i.colorSource = telem.useSource(ctx, colorSource, i.colorSource);
    this.updateState();
    this.stopListeningStatus?.();
    this.stopListeningColor?.();
    this.stopListeningStatus = i.statusSource.onChange(() => {
      this.updateState();
    });
    this.stopListeningColor = i.colorSource.onChange(() => {
      this.updateState();
    });
  }

  afterDelete(): void {
    this.internal.statusSource.cleanup?.();
    this.internal.colorSource.cleanup?.();
  }

  render(): void {}

  updateState(): void {
    const colorVal = this.internal.colorSource.value();
    const status = this.internal.statusSource.value();
    if (
      color.equals(colorVal, this.state.color) &&
      status.message === this.state.status.message
    )
      return;
    this.setState((p) => ({ ...p, color: colorVal, status }));
  }
}
