// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { type notation } from "@synnaxlabs/x";

import { telem } from "@/telem/aether";

export interface StringSourceArgs {
  channel?: channel.Key;
  rollingAverage?: number;
  precision?: number;
  notation?: notation.Notation;
}

/** stringSource builds the formatted display pipeline for a value channel. */
export const stringSource = ({
  channel = 0,
  rollingAverage = 1,
  precision = 2,
  notation,
}: StringSourceArgs): telem.StringSourceSpec =>
  telem.sourcePipeline("string", {
    connections: [
      { from: "valueStream", to: "rollingAverage" },
      { from: "rollingAverage", to: "stringifier" },
    ],
    segments: {
      valueStream: telem.streamChannelValue({ channel }),
      rollingAverage: telem.rollingAverage({ windowSize: rollingAverage }),
      stringifier: telem.stringifyNumber({ precision, notation }),
    },
    outlet: "stringifier",
  });
