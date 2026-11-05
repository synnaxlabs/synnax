// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type project, type video } from "@synnaxlabs/client";
import { uuid, verbs } from "@synnaxlabs/x";

import { Flux } from "@/flux";
import { Scope } from "@/video/scope";

const RESOURCE_NAME = "video view";

export type UseDeleteParams = video.Key | video.Key[];

export type RetrieveQuery = video.RetrieveSingleParams;

export const { use, useEnsure, useTombstone, createSelector } = Flux.createRetrieve<
  RetrieveQuery,
  video.Video
>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.videos.retrieve(query),
  onChange: ({ client, query }, handler) => client.videos.onChange(query, handler),
  getCached: ({ client, query }) => client.videos.getCached(query),
  awaitCreation: true,
});

export const useName = Scope.bindHook(createSelector(({ name }) => name));

export const useChannel = Scope.bindHook(createSelector(({ channel }) => channel));

export const useRange = Scope.bindHook(createSelector(({ range }) => range));

export const {
  useUndo: useUndoBase,
  useRedo: useRedoBase,
  useSingleDispatch: useSingleDispatchBase,
} = Flux.createDispatch<video.Key, video.Video, video.Action>({
  domain: (client) => client.videos,
});

export const useSingleDispatch = Scope.bindHook(useSingleDispatchBase);
export const useUndo = Scope.bindHook(useUndoBase);
export const useRedo = Scope.bindHook(useRedoBase);

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.videos.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export interface CreateParams extends video.New {
  project?: project.Key;
}

export const { useUpdate: useCreate } = Flux.createUpdate<CreateParams, video.Video>({
  name: RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ client, data, onOptimisticComplete }) =>
    await client.videos.create(data.project ?? uuid.ZERO, data, {
      onOptimistic: async ([optimistic]) => await onOptimisticComplete(optimistic),
    }),
});

export interface RenameParams extends Pick<video.Video, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, name } = data;
    await onOptimisticComplete(data);
    await client.videos.rename(key, name);
    return data;
  },
});
