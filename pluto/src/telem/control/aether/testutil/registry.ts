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
import { control } from "@/telem/control/aether";

const chipStateZ = z.object({ sink: z.unknown() }).loose();
const indicatorStateZ = z.object({}).loose();

/**
 * Returns a registry that mounts stubs in place of the control chip and indicator, so a
 * spec needs no controller.
 * @param onChipSink - Receives the sink that the chip acquires control with, on every
 * update of the chip.
 */
export const createRegistry = (
  onChipSink?: (sink: unknown) => void,
): aether.ComponentRegistry => {
  class Chip extends aether.Leaf<typeof chipStateZ> {
    override schema = chipStateZ;

    override afterUpdate(): void {
      onChipSink?.(this.state.sink);
    }
  }
  class Indicator extends aether.Leaf<typeof indicatorStateZ> {
    override schema = indicatorStateZ;
  }
  return { [control.Chip.TYPE]: Chip, [control.Indicator.TYPE]: Indicator };
};
