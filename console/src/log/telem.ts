// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { telem } from "@synnaxlabs/pluto";
import { primitive, TimeSpan } from "@synnaxlabs/x";

/**
 * Creates a telemetry source that combines multiple channel streams into a single
 * MultiSeries for display in the log component.
 */
export const createMultiChannelLogSource = (
  channels: channel.Key[],
  timeSpan: TimeSpan,
  keepFor: TimeSpan,
): telem.SeriesSourceSpec => {
  // Filter out zero channels
  const validChannels = channels.filter((ch) => !primitive.isZero(ch));

  // If no valid channels, return noop
  if (validChannels.length === 0) return telem.noopSeriesSourceSpec;

  // Use the new multi-channel streaming source
  return telem.streamMultiChannelData({
    channels: validChannels,
    timeSpan,
    keepFor,
  });
};
