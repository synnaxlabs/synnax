// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { type z } from "zod/v4";

import { Query } from "@/query";

export const useCommandSynchronizer = (
  onCommandUpdate: (command: task.Command) => void,
): void =>
  Query.useParsedListener(task.COMMAND_CHANNEL_NAME, task.commandZ, onCommandUpdate);

export const useStateSynchronizer = (
  onStateUpdate: (state: task.Status) => void,
): void =>
  Query.useParsedListener(task.STATE_CHANNEL_NAME, task.statusZ(), onStateUpdate);

export const useSetSynchronizer = (onSet: (key: task.Key) => void): void =>
  Query.useParsedListener(task.SET_CHANNEL_NAME, task.keyZ, onSet);

export const useDeleteSynchronizer = (onDelete: (key: task.Key) => void): void =>
  Query.useParsedListener(task.DELETE_CHANNEL_NAME, task.keyZ, onDelete);

const baseUse = <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
>(
  key: task.Key | undefined,
  schemas: task.Schemas<Type, Config, StatusData>,
) => {
  const useQuery = Query.create<
    task.Key | undefined,
    task.Task<Type, Config, StatusData> | null
  >({
    name: "Task",
    queryFn: async ({ client, params: key }) => {
      if (key == null) return null;
      return await client.hardware.tasks.retrieve({ key, schemas });
    },
    listeners: [
      {
        channel: task.SET_CHANNEL_NAME,
        onChange: Query.parsedHandler(
          task.keyZ,
          async ({ client, changed, params: key, onChange }) => {
            if (key == null || changed.toString() !== key.toString()) return;
            onChange(await client.hardware.tasks.retrieve({ key, schemas }));
          },
        ),
      },
    ],
  });
  return useQuery(key);
};

export const use = baseUse;
