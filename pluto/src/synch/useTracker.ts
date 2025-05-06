// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { useEffect } from "react";
import { type z } from "zod";

import { useAddListener } from "@/synch/useAddListener";

export const useTracker = <Z extends z.ZodTypeAny>(
  channel: channel.Name,
  schema: Z,
  onUpdate: (value: z.output<Z>) => void,
): void => {
  const addListener = useAddListener();
  useEffect(
    () =>
      addListener({
        channels: channel,
        handler: (frame) => frame.get(channel).parseJSON(schema).forEach(onUpdate),
      }),
    [addListener, channel, schema, onUpdate],
  );
};
