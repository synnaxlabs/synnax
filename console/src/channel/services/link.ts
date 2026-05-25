// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { LinePlot } from "@/lineplot";
import { type Link } from "@/link";

export const handleLink: Link.Handler = async ({ client, key, placeLayout }) => {
  const channel = await client.channels.retrieve(key);
  placeLayout(
    LinePlot.create({
      name: `${channel.name} Plot`,
      pendingUpload: {
        channels: { x1: 0, x2: 0, y1: [channel.key], y2: [], y3: [], y4: [] },
      },
    }),
  );
};
