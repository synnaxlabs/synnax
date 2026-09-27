// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { TimeSpan } from "@synnaxlabs/x";

import { telem } from "@/telem/aether";

export type Redline = schematic.Redline;
export type Band = schematic.Band;

/** How long a flashing band holds each of its two fills. */
const FLASH_PERIOD = TimeSpan.milliseconds(500);

/**
 * Builds the background color telemetry for a value painted through the given
 * redline.
 * @param source - The value's display telemetry. Its text is read as a number.
 * @returns The color telemetry, or undefined when the redline paints nothing.
 */
export const backgroundTelem = (
  source: telem.StringSourceSpec,
  { bands, base, smooth }: Redline,
): telem.ColorSourceSpec | undefined => {
  if (bands.length === 0 && base == null) return undefined;
  const flashing = bands.some((band) => band.flashing);
  return telem.sourcePipeline("color", {
    connections: [
      { from: "source", to: "band" },
      ...(flashing ? [{ from: "phase", to: "band" }] : []),
    ],
    segments: {
      source,
      band: telem.bandColor({ bands, base, smooth }),
      ...(flashing ? { phase: telem.clock({ period: FLASH_PERIOD }) } : {}),
    },
    outlet: "band",
  });
};
