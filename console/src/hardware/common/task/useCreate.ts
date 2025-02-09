// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { rack, type task } from "@synnaxlabs/client";
import { Synnax } from "@synnaxlabs/pluto";
import { type UnknownRecord } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";

export const useCreate = <
  Config extends UnknownRecord,
  Details extends {} = UnknownRecord,
  Type extends string = string,
>(
  layoutKey: string,
) => {
  const client = Synnax.use();
  const dispatch = useDispatch();
  return useCallback(
    async (task: task.New<Config, Type>, rackKey: rack.Key) => {
      if (client == null) throw new Error("Client not found");
      const rck = await client.hardware.racks.retrieve(
        rackKey ?? rack.DEFAULT_CHANNEL_NAME,
      );
      const createdTask = await rck.createTask<Config, Details, Type>(task);
      dispatch(Layout.setArgs({ key: layoutKey, args: { taskKey: createdTask.key } }));
      return createdTask;
    },
    [client?.key, dispatch, layoutKey],
  );
};
