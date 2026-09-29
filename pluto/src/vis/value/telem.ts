// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";

import { telem } from "@/telem/aether";

export interface NumberSourceArgs {
  channel?: channel.Key;
  rollingAverage?: number;
}

/** numberSource builds the averaged numeric pipeline for a value channel. */
export const numberSource = ({
  channel = 0,
  rollingAverage = 1,
}: NumberSourceArgs): telem.NumberSourceSpec =>
  telem.sourcePipeline("number", {
    connections: [{ from: "valueStream", to: "rollingAverage" }],
    segments: {
      valueStream: telem.streamChannelValue({ channel }),
      rollingAverage: telem.rollingAverage({ windowSize: rollingAverage }),
    },
    outlet: "rollingAverage",
  });
