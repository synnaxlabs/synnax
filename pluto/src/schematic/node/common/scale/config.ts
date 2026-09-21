// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { bounds, color, location, notation, text } from "@synnaxlabs/x";
import { z } from "zod";

import { telem } from "@/telem/aether";
import { Staleness } from "@/vis/staleness";

/** Side the ticks and the readout sit on until the user moves them. */
export const DEFAULT_SIDE: location.Outer = "right";

/** Stored shape of a live scale indicator, shared by every symbol that renders one. */
export const configZ = z.object({
  ...Staleness.configZ.shape,
  telem: telem.numberSourceSpecZ.optional(),
  bounds: bounds.boundsZ().default(() => bounds.construct(0, 100)),
  color: color.crudeZ.default(color.ZERO),
  axisColor: color.crudeZ.default(color.ZERO),
  textColor: color.crudeZ.default(color.ZERO),
  units: z.string().default(""),
  notation: notation.notationZ.default("standard"),
  precision: z.number().default(2),
  showFill: z.boolean().default(true),
  showCaret: z.boolean().default(true),
  showScale: z.boolean().default(true),
  side: location.outerZ.default(DEFAULT_SIDE),
  caretSide: location.outerZ.default(DEFAULT_SIDE),
  level: text.levelZ.default("small"),
});
export type Config = z.infer<typeof configZ>;

export const defaultConfig = (overrides: Partial<Config> = {}): Config =>
  configZ.parse({ ...Staleness.ZERO_CONFIG, ...overrides });

const CONNECTIONS: telem.Connection[] = [{ from: "valueStream", to: "rollingAverage" }];

export interface TelemProps {
  channel: channel.Key;
  windowSize: number;
}

const DEFAULT_TELEM_PROPS: TelemProps = { channel: 0, windowSize: 1 };

export const createTelem = (
  props: Partial<TelemProps> = {},
): telem.NumberSourceSpec => {
  const { channel, windowSize } = { ...DEFAULT_TELEM_PROPS, ...props };
  return telem.sourcePipeline("number", {
    connections: CONNECTIONS,
    segments: {
      valueStream: telem.streamChannelValue({ channel }),
      rollingAverage: telem.rollingAverage({ windowSize }),
    },
    outlet: "rollingAverage",
  });
};

/**
 * Reads back the properties a spec was built from. A spec that does not match the
 * pipeline the form writes falls back to the defaults, so an unrecognized shape edits
 * cleanly instead of throwing.
 */
export const parseTelem = (spec?: telem.NumberSourceSpec): TelemProps => {
  const pipeline = telem.sourcePipelinePropsZ.safeParse(spec?.props);
  if (!pipeline.success) return DEFAULT_TELEM_PROPS;
  const { segments } = pipeline.data;
  const stream = telem.streamChannelValuePropsZ.safeParse(segments.valueStream?.props);
  const average = telem.rollingAverageProps.safeParse(segments.rollingAverage?.props);
  return {
    channel:
      stream.success && typeof stream.data.channel === "number"
        ? stream.data.channel
        : DEFAULT_TELEM_PROPS.channel,
    windowSize: average.success
      ? (average.data.windowSize ?? DEFAULT_TELEM_PROPS.windowSize)
      : DEFAULT_TELEM_PROPS.windowSize,
  };
};
