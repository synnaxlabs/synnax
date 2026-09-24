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
import { useEffect } from "react";

import { Aether } from "@/aether";
import { latestSample } from "@/vis/latestSample/aether";

export interface UseProps {
  aetherKey?: string;
  channel: channel.Key;
}

/** Undefined until the first read settles, null when the channel has no samples. */
export const use = ({ aetherKey, channel }: UseProps): TimeStamp | null | undefined => {
  const [, { time }, setState] = Aether.use({
    aetherKey,
    type: latestSample.LatestSample.TYPE,
    schema: latestSample.stateZ,
    initialState: { channel },
  });
  useEffect(() => setState((s) => ({ ...s, channel })), [channel, setState]);
  return time;
};
