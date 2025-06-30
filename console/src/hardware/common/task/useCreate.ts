// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack, type task } from "@synnaxlabs/client";
import { Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { useDispatch } from "react-redux";
import { type z } from "zod/v4";

import { NULL_CLIENT_ERROR } from "@/errors";
import { Layout } from "@/layout";

export const useCreate = <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
>(
  layoutKey: string,
  schemas: task.Schemas<Type, Config, StatusData>,
) => {
  const client = Synnax.use();
  const dispatch = useDispatch();
  return useCallback(
    async (task: task.New<Type, Config>, rackKey: rack.Key) => {
      if (client == null) throw NULL_CLIENT_ERROR;
      const rck = await client.hardware.racks.retrieve(rackKey);
      const createdTask = await rck.createTask(task, schemas);
      dispatch(Layout.setArgs({ key: layoutKey, args: { taskKey: createdTask.key } }));
      dispatch(Layout.setAltKey({ key: layoutKey, altKey: createdTask.key }));
      return createdTask;
    },
    [client?.key, dispatch, layoutKey],
  );
};
