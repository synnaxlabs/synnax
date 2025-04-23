// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { type z } from "zod";

import { useAsyncEffect } from "@/hooks";
import { useAddListener } from "@/synch/useAddListener";
import { Synnax } from "@/synnax";

export const useStateChannel = <Z extends z.ZodTypeAny>(
  channel: channel.Name,
  schema: Z,
  onStateUpdate: (state: z.output<Z>) => void,
): void => {
  const client = Synnax.use();
  const addListener = useAddListener();
  useAsyncEffect(async () => {
    if (client == null) return;
    return addListener({
      channels: channel,
      handler: (frame) => frame.get(channel).parseJSON(schema).forEach(onStateUpdate),
    });
  }, [client, addListener, channel, schema, onStateUpdate]);
};
