// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, log, NotFoundError, type project } from "@synnaxlabs/client";
import { array, compare, uuid } from "@synnaxlabs/x";

import { Flux } from "@/flux";
import { Scope } from "@/log/scope";
import { Ontology } from "@/ontology";

export const FLUX_STORE_KEY = "logs";
const RESOURCE_NAME = "log";

export interface FluxStore extends Flux.UndoableUnaryStore<
  log.Key,
  log.Log,
  log.Action
> {}

export type UseDeleteParams = log.Key | log.Key[];

export interface FluxSubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
}

export type RetrieveQuery = log.RetrieveSingleParams;

export const retrieveSingle = async ({
  store,
  client,
  query: { key },
}: Flux.RetrieveParams<RetrieveQuery, FluxSubStore>) => {
  const cached = store.logs.get(key);
  if (cached != null) return cached;
  const l = await client.logs.retrieve({ key });
  store.logs.set(l);
  return l;
};

export const {
  useRetrieve,
  useRetrieveObservable,
  useRetrieveSuspended,
  useEnsureRetrieved,
} = Flux.createRetrieve<RetrieveQuery, log.Log, FluxSubStore>({
  name: RESOURCE_NAME,
  retrieve: retrieveSingle,
  retrieveCached: ({ store, query: { key } }) => store.logs.get(key),
  mountListeners: ({ store, query: { key }, onChange }) =>
    store.logs.onSet(onChange, key),
});

export interface SelectKeyParams {
  key: log.Key;
}

const requireLog = (store: FluxSubStore, key: log.Key): log.Log => {
  const l = store.logs.get(key);
  if (l == null) throw new NotFoundError(`Log with key ${key} not found`);
  return l;
};

export const [useSelectName, useGetName] = Scope.bindSelector(
  Flux.createSelector<FluxSubStore, SelectKeyParams, string>({
    subscribe: (store, { key }, notify) => store.logs.onSet(notify, key),
    select: (store, { key }) => requireLog(store, key).name,
  }),
);

export const [useSelectChannels, useGetChannels] = Scope.bindSelector(
  Flux.createSelector<FluxSubStore, SelectKeyParams, log.ChannelEntry[]>({
    subscribe: (store, { key }, notify) => store.logs.onSet(notify, key),
    select: (store, { key }) => requireLog(store, key).channels,
  }),
);

// useSelectChannelKeys returns the ordered channel keys, re-rendering only when the
// set or order of channels changes — not when an individual entry's display config
// is edited. Iterate the toolbar list off this and read each row via
// useSelectChannelEntry.
export const [useSelectChannelKeys, useGetChannelKeys] = Scope.bindSelector(
  Flux.createSelector<FluxSubStore, SelectKeyParams, channel.Key[]>({
    subscribe: (store, { key }, notify) => store.logs.onSet(notify, key),
    select: (store, { key }) => requireLog(store, key).channels.map((e) => e.channel),
    equal: compare.arraysEqual,
  }),
);

export interface SelectChannelEntryParams extends SelectKeyParams {
  channel: channel.Key;
}

// useSelectChannelEntry returns the entry for a single channel, or null when the
// channel has no entry. reduceAll's structural sharing keeps the entry reference
// stable across dispatches that don't touch it, so editing one channel does not
// re-render the rows of the others.
export const [useSelectChannelEntry, useGetChannelEntry] = Scope.bindSelector(
  Flux.createSelector<FluxSubStore, SelectChannelEntryParams, log.ChannelEntry | null>({
    subscribe: (store, { key }, notify) => store.logs.onSet(notify, key),
    select: (store, { key, channel }) =>
      requireLog(store, key).channels.find((e) => e.channel === channel) ?? null,
  }),
);

export const [useSelectTimestampPrecision, useGetTimestampPrecision] =
  Scope.bindSelector(
    Flux.createSelector<FluxSubStore, SelectKeyParams, number>({
      subscribe: (store, { key }, notify) => store.logs.onSet(notify, key),
      select: (store, { key }) => requireLog(store, key).timestampPrecision,
    }),
  );

export const [useSelectHideChannelNames, useGetHideChannelNames] = Scope.bindSelector(
  Flux.createSelector<FluxSubStore, SelectKeyParams, boolean>({
    subscribe: (store, { key }, notify) => store.logs.onSet(notify, key),
    select: (store, { key }) => requireLog(store, key).hideChannelNames,
  }),
);

export const [useSelectHideReceiptTimestamp, useGetHideReceiptTimestamp] =
  Scope.bindSelector(
    Flux.createSelector<FluxSubStore, SelectKeyParams, boolean>({
      subscribe: (store, { key }, notify) => store.logs.onSet(notify, key),
      select: (store, { key }) => requireLog(store, key).hideReceiptTimestamp,
    }),
  );

// kindOfTransaction keys single-channel edits by channel so rapid edits to one
// channel's config coalesce into a single undoable, while edits to distinct
// channels stay separate. Log-level edits (name, precision, flags) key by their
// action type; multi-action batches collapse into one "transaction" undoable.
const kindOfTransaction = (actions: log.Action[]): string => {
  if (actions.length === 0) return "default";
  if (actions.length > 1) return "transaction";
  const a = actions[0];
  switch (a.type) {
    case "add_channel":
      return `channel:${a.addChannel.channel}`;
    case "remove_channel":
      return `channel:${a.removeChannel.channel}`;
    case "set_channel_entry":
      return `channel:${a.setChannelEntry.entry.channel}`;
    case "set_channel_color":
      return `channel:${a.setChannelColor.channel}`;
    case "set_channel_notation":
      return `channel:${a.setChannelNotation.channel}`;
    case "set_channel_precision":
      return `channel:${a.setChannelPrecision.channel}`;
    case "set_channel_alias":
      return `channel:${a.setChannelAlias.channel}`;
    case "set_channel_timestamp_format":
      return `channel:${a.setChannelTimestampFormat.channel}`;
    case "set_channel_timestamp_tz":
      return `channel:${a.setChannelTimestampTz.channel}`;
    default:
      return a.type;
  }
};

export const FLUX_STORE_CONFIG = Flux.createUndoableStore<
  log.Key,
  log.Log,
  log.Action,
  typeof FLUX_STORE_KEY,
  FluxSubStore
>({
  storeKey: FLUX_STORE_KEY,
  reduce: log.reduceAll,
  channel: log.SET_CHANNEL_NAME,
  schema: log.scopedActionZ,
  kindOf: kindOfTransaction,
});

export const {
  useDispatch,
  useUndo: useUndoBase,
  useRedo: useRedoBase,
  useSingleDispatch: useSingleDispatchBase,
} = Flux.createDispatch<
  log.Key,
  log.Log,
  log.Action,
  typeof FLUX_STORE_KEY,
  FluxSubStore
>({
  storeKey: FLUX_STORE_KEY,
  send: ({ client, key, actions, dispatchKey }) =>
    client.logs.dispatch(key, dispatchKey, actions),
});

export const useSingleDispatch = Scope.bindHook(useSingleDispatchBase);
export const useUndo = Scope.bindHook(useUndoBase);
export const useRedo = Scope.bindHook(useRedoBase);

export const { useUpdate: useDelete } = Flux.createUpdate<
  UseDeleteParams,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, rollbacks, store, onOptimisticComplete }) => {
    const keys = array.toArray(data);
    const ids = log.ontologyID(keys);
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.push(store.relationships.delete(relFilter));
    rollbacks.push(store.logs.delete(keys));
    await onOptimisticComplete(data);
    await client.logs.delete(data);
    return data;
  },
});

export interface CreateParams extends log.New {
  project?: project.Key;
}

export const { useUpdate: useCreate } = Flux.createUpdate<
  CreateParams,
  FluxSubStore,
  log.Log
>({
  name: RESOURCE_NAME,
  verbs: Flux.CREATE_VERBS,
  update: async ({ client, data, store, rollbacks, onOptimisticComplete }) => {
    const optimistic = log.logZ.parse(data);
    rollbacks.push(store.logs.set(optimistic));
    await onOptimisticComplete(optimistic);
    const project = data.project ?? uuid.ZERO;
    const created = await client.logs.create(project, optimistic);
    store.logs.set(created);
    return created;
  },
});

export interface RenameParams extends Pick<log.Log, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, rollbacks, store, onOptimisticComplete }) => {
    const { key, name } = data;
    rollbacks.push(Flux.partialUpdate(store.logs, key, { name }));
    rollbacks.push(Ontology.renameFluxResource(store, log.ontologyID(key), name));
    await onOptimisticComplete(data);
    await client.logs.rename(key, data.name);
    return data;
  },
});
