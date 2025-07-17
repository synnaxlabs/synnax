// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { status } from "@synnaxlabs/x";
import { z } from "zod";

import { Flux } from "@/flux";
import { Sync } from "@/flux/sync";

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

interface QueryParams extends Flux.Params {
  key: task.Key | undefined;
}

export const createRetrieveQuery = <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
>(
  schemas: task.Schemas<Type, Config, StatusData>,
) =>
  Flux.createRetrieve<QueryParams, task.Task<Type, Config, StatusData> | null>({
    name: "Task",
    retrieve: async ({ client, params: { key } }) => {
      if (key == null) return null;
      return await client.hardware.tasks.retrieve({ key, schemas });
    },
    listeners: [
      {
        channel: task.SET_CHANNEL_NAME,
        onChange: Sync.parsedHandler(
          task.keyZ,
          async ({ client, changed, onChange, params: { key } }) => {
            if (key == null || changed.toString() !== key.toString()) return;
            onChange(await client.hardware.tasks.retrieve({ key, schemas }));
          },
        ),
      },
    ],
  });

export interface ListParams extends Flux.Params {
  term?: string;
  offset?: number;
  limit?: number;
}

export const useList = Flux.createList<ListParams, task.Key, task.Task>({
  name: "Task",
  retrieve: async ({ client, params }) =>
    await client.hardware.tasks.retrieve({
      includeStatus: true,
      ...params,
    }),
  retrieveByKey: async ({ client, key }) =>
    await client.hardware.tasks.retrieve({ key }),
  listeners: [
    {
      channel: task.SET_CHANNEL_NAME,
      onChange: Sync.parsedHandler(
        task.keyZ,
        async ({ client, changed: key, onChange }) =>
          onChange(key, await client.hardware.tasks.retrieve({ key })),
      ),
    },
    {
      channel: task.DELETE_CHANNEL_NAME,
      onChange: Sync.parsedHandler(task.keyZ, async ({ changed, onDelete }) =>
        onDelete(changed),
      ),
    },
    {
      channel: task.STATUS_CHANNEL_NAME,
      onChange: Sync.parsedHandler(
        task.statusZ(z.unknown()),
        async ({ changed, onChange, client }) => {
          onChange(changed.details.task, (prev) => {
            if (prev == null) return prev;
            return client.hardware.tasks.sugar({ ...prev, status: changed });
          });
        },
      ),
    },
    {
      channel: task.COMMAND_CHANNEL_NAME,
      onChange: Sync.parsedHandler(
        task.commandZ,
        async ({ changed, onChange, client }) => {
          onChange(changed.task, (prev) => {
            if (prev == null) return prev;
            return client.hardware.tasks.sugar({
              ...prev,
              status: status.create<task.StatusDetails<z.ZodUnknown>>({
                variant: "loading",
                message: "Executing command...",
                details: { task: changed.task, running: true, data: {} },
              }),
            });
          });
        },
      ),
    },
  ],
});
