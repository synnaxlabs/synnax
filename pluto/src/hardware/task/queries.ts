// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { useEffect } from "react";
import { z } from "zod";

import { Flux } from "@/flux";

export interface FluxStore extends Flux.UnaryStore<task.Key, task.Task> {}

interface SubStore extends Flux.Store {
  tasks: FluxStore;
}

// Temporary hack that filters the set of commands that should change the
// status of a task to loading.
// Issue: https://linear.app/synnax/issue/SY-2723/fix-handling-of-non-startstop-commands-loading-indicators-in-tasks
const LOADING_COMMANDS = ["start", "stop"];

const SET_LISTENER: Flux.ChannelListener<SubStore, typeof task.keyZ> = {
  channel: task.SET_CHANNEL_NAME,
  schema: task.keyZ,
  onChange: async ({ store, changed, client }) => {
    const t = await client.hardware.tasks.retrieve({ key: changed });
    store.tasks.set(changed, t);
  },
};

const DELETE_LISTENER: Flux.ChannelListener<SubStore, typeof task.keyZ> = {
  channel: task.DELETE_CHANNEL_NAME,
  schema: task.keyZ,
  onChange: async ({ store, changed }) => store.tasks.delete(changed),
};

const unknownStatusZ = task.statusZ(z.unknown());

const SET_STATUS_LISTENER: Flux.ChannelListener<SubStore, typeof unknownStatusZ> = {
  channel: task.STATUS_CHANNEL_NAME,
  schema: unknownStatusZ,
  onChange: async ({ store, changed }) =>
    store.tasks.set(changed.details.task, (prev) =>
      prev == null ? prev : ({ ...prev, status: changed } as task.Task),
    ),
};

const SET_COMMAND_LISTENER: Flux.ChannelListener<SubStore, typeof task.commandZ> = {
  channel: task.COMMAND_CHANNEL_NAME,
  schema: task.commandZ,
  onChange: async ({ store, changed, client }) =>
    store.tasks.set(changed.task, (prev) => {
      if (prev == null || !LOADING_COMMANDS.includes(changed.type)) return prev;
      return client.hardware.tasks.sugar({
        ...prev,
        status: {
          ...prev.status,
          variant: "loading",
          message: "Executing command...",
          details: { task: changed.task, running: true, data: {} },
        },
      } as task.Task);
    }),
};

export const useStatusSynchronizer = (
  onStatus: (status: task.Status) => void,
): void => {
  const store = Flux.useStore<SubStore>();
  useEffect(
    () =>
      store.tasks.onSet(async (task) => {
        if (task.status == null) return;
        onStatus(task.status);
      }),
    [store],
  );
};

export const STORE_CONFIG: Flux.UnaryStoreConfig<SubStore> = {
  listeners: [SET_LISTENER, DELETE_LISTENER, SET_STATUS_LISTENER, SET_COMMAND_LISTENER],
};

interface QueryParams {
  key: task.Key | undefined;
}

export const createRetrieveQuery = <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
>(
  schemas: task.Schemas<Type, Config, StatusData>,
) =>
  Flux.createRetrieve<
    QueryParams,
    task.Task<Type, Config, StatusData> | null,
    SubStore
  >({
    name: "Task",
    retrieve: async ({ client, params: { key } }) => {
      if (key == null) return null;
      return await client.hardware.tasks.retrieve({
        key,
        includeStatus: true,
        schemas,
      });
    },
    mountListeners: ({ store, params: { key }, onChange }) => [
      store.tasks.onSet(async (task) => {
        if (key == null || task.key !== key) return;
        onChange(task as unknown as task.Task<Type, Config, StatusData>);
      }, key),
    ],
  });

export interface ListParams {
  term?: string;
  offset?: number;
  limit?: number;
}

export const useList = Flux.createList<ListParams, task.Key, task.Task, SubStore>({
  name: "Task",
  retrieve: async ({ client, params }) =>
    await client.hardware.tasks.retrieve({
      includeStatus: true,
      ...params,
    }),
  retrieveByKey: async ({ client, key }) =>
    await client.hardware.tasks.retrieve({ key }),
  mountListeners: ({ store, onChange, onDelete }) => [
    store.tasks.onSet(async (task) => {
      onChange(task.key, task);
    }),
    store.tasks.onDelete(async (key) => onDelete(key)),
  ],
});
