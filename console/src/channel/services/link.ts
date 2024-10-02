// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { create as createLinePlot } from "@/lineplot/LinePlot";
import { ZERO_CHANNELS_STATE } from "@/lineplot/slice";
import { Link } from "@/link";

export const linkHandler: Link.Handler = async ({
  resource,
  resourceKey,
  client,
  placer,
  addStatus,
}): Promise<boolean> => {
  if (resource != "channel") return false;
  try {
    const channel = await client.channels.retrieve(resourceKey);
    placer(
      createLinePlot({
        channels: {
          ...ZERO_CHANNELS_STATE,
          y1: [channel.key],
        },
      }),
    );
  } catch (e) {
    addStatus({
      variant: "error",
      message: (e as Error).message,
    });
  }
  return true;
};
