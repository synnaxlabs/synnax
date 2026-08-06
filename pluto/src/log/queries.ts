// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type log, type project } from "@synnaxlabs/client";
import { compare, uuid, verbs } from "@synnaxlabs/x";

import { Flux } from "@/flux";
import { Scope } from "@/log/scope";

const RESOURCE_NAME = "log";

export type UseDeleteParams = log.Key | log.Key[];

export type RetrieveQuery = log.RetrieveSingleParams;

export const { use, useEnsure, useTombstone, createSelector } = Flux.createRetrieve<
  RetrieveQuery,
  log.Log
>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.logs.retrieve(query),
  onChange: ({ client, query }, handler) => client.logs.onChange(query, handler),
  getCached: ({ client, query }) => client.logs.getCached(query),
});

export interface SelectKeyParams {
  key: log.Key;
}

export const useSelectName = Scope.bindHook(createSelector(({ name }) => name));

export const useSelectChannels = Scope.bindHook(
  createSelector(({ channels }) => channels),
);

// useSelectChannelKeys returns the ordered channel keys, re-rendering only
// when the set or order of channels changes, not when an entry's display
// config is edited. Iterate the toolbar list off this and read each row via
// useSelectChannelEntry.
export const useSelectChannelKeys = Scope.bindHook(
  createSelector(
    ({ channels }) => channels.map((e) => e.channel),
    (a, b) => compare.arraysEqual(a, b),
  ),
);

export interface SelectChannelEntryParams extends SelectKeyParams {
  channel: channel.Key;
}

// useSelectChannelEntry returns the entry for a single channel, or null when
// the channel has no entry. Structural sharing keeps the entry reference
// stable across dispatches that don't touch it, so editing one channel does
// not re-render the rows of the others.
export const useSelectChannelEntry = Scope.bindHook(
  createSelector<log.ChannelEntry | null, SelectChannelEntryParams>(
    ({ channels }, { channel }) => channels.find((e) => e.channel === channel) ?? null,
  ),
);

export const useSelectTimestampPrecision = Scope.bindHook(
  createSelector(({ timestampPrecision }) => timestampPrecision),
);

export const useSelectHideChannelNames = Scope.bindHook(
  createSelector(({ hideChannelNames }) => hideChannelNames),
);

export const useSelectHideReceiptTimestamp = Scope.bindHook(
  createSelector(({ hideReceiptTimestamp }) => hideReceiptTimestamp),
);

export const {
  useDispatch,
  useUndo: useUndoBase,
  useRedo: useRedoBase,
  useSingleDispatch: useSingleDispatchBase,
} = Flux.createDispatch<log.Key, log.Log, log.Action>({
  domain: (client) => client.logs,
});

export const useSingleDispatch = Scope.bindHook(useSingleDispatchBase);
export const useUndo = Scope.bindHook(useUndoBase);
export const useRedo = Scope.bindHook(useRedoBase);

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.logs.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export interface CreateParams extends log.New {
  project?: project.Key;
}

export const { useUpdate: useCreate } = Flux.createUpdate<CreateParams, log.Log>({
  name: RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ client, data, onOptimisticComplete }) =>
    await client.logs.create(data.project ?? uuid.ZERO, data, {
      onOptimistic: async ([optimistic]) => await onOptimisticComplete(optimistic),
    }),
});

export interface RenameParams extends Pick<log.Log, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, name } = data;
    await onOptimisticComplete(data);
    await client.logs.rename(key, name);
    return data;
  },
});
