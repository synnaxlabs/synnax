// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { type TimeStamp } from "@synnaxlabs/x";

import { Aether } from "@/aether";
import { telem } from "@/telem/aether";
import { latestSample } from "@/vis/latestSample/aether";

export interface UseProps extends Aether.ComponentProps {
  /** Read on the mounting render only. Remount under a React key to change it. */
  channel: channel.Key;
}

/** @returns the time of the channel's newest sample, null until one is known. */
export const use = ({ aetherKey, channel }: UseProps): TimeStamp | null => {
  const [, { time }] = Aether.use({
    aetherKey,
    type: latestSample.LatestSample.TYPE,
    schema: latestSample.stateZ,
    initialState: { source: telem.streamChannelValue({ channel }) },
  });
  return time;
};
