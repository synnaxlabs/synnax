// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { z } from "zod/v4";

import { Query } from "@/query";
import { Sync } from "@/query/sync";

export const useCommandSynchronizer = (
  onCommandUpdate: (command: task.Command) => void,
): void =>
  Sync.useListener({
    channel: task.COMMAND_CHANNEL_NAME,
    onChange: Sync.parsedHandler(task.commandZ, async (args) => {
      onCommandUpdate(args.changed);
    }),
  });

export const useStatusSynchronizer = <StatusData extends z.ZodType>(
  onStatusUpdate: (status: task.Status<StatusData>) => void,
  statusDataZ: StatusData = z.unknown() as unknown as StatusData,
): void =>
  Sync.useListener({
    channel: task.STATUS_CHANNEL_NAME,
    onChange: Sync.parsedHandler(task.statusZ(statusDataZ), async (args) => {
      onStatusUpdate(args.changed);
    }),
  });

export const useSetSynchronizer = (onSet: (key: task.Key) => void): void =>
  Sync.useListener({
    channel: task.SET_CHANNEL_NAME,
    onChange: Sync.parsedHandler(task.keyZ, async (args) => {
      onSet(args.changed);
    }),
  });

export const useDeleteSynchronizer = (onDelete: (key: task.Key) => void): void =>
  Sync.useListener({
    channel: task.DELETE_CHANNEL_NAME,
    onChange: Sync.parsedHandler(task.keyZ, async (args) => {
      onDelete(args.changed);
    }),
  });

interface QueryParams extends Query.Params {
  key: task.Key | undefined;
}

export const use = <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
>(
  key: task.Key | undefined,
  schemas: task.Schemas<Type, Config, StatusData>,
) =>
  Query.use<QueryParams, task.Task<Type, Config, StatusData> | null>({
    name: "Task",
    params: { key },
    retrieve: async ({ client, params: { key } }) => {
      if (key == null) return null;
      return await client.hardware.tasks.retrieve({ key, schemas });
    },
    listeners: [
      {
        channel: task.SET_CHANNEL_NAME,
        onChange: Sync.parsedHandler(
          task.keyZ,
          async ({ client, changed, onChange }) => {
            if (key == null || changed.toString() !== key.toString()) return;
            onChange(await client.hardware.tasks.retrieve({ key, schemas }));
          },
        ),
      },
    ],
  });
