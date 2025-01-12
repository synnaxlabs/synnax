// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";

import { LinePlot } from "@/lineplot";
import { type Link } from "@/link";

export const linkHandler: Link.Handler = async ({
  resource,
  resourceKey,
  client,
  place,
  addStatus,
}): Promise<boolean> => {
  if (resource !== channel.ONTOLOGY_TYPE) return false;
  try {
    const channel = await client.channels.retrieve(resourceKey);
    place(
      LinePlot.create({
        channels: {
          ...LinePlot.ZERO_CHANNELS_STATE,
          y1: [channel.key],
        },
        name: `${channel.name} Plot`,
      }),
    );
  } catch (e) {
    if (!(e instanceof Error)) throw e;
    addStatus({
      variant: "error",
      description: "Failed to open channel from URL",
      message: e.message,
    });
  }
  return true;
};
