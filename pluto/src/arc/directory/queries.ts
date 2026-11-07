// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc, type resource } from "@synnaxlabs/client";
import { verbs } from "@synnaxlabs/x";

import { Flux } from "@/flux";

export const RESOURCE_NAME = "directory";

export interface CreateParams extends arc.directory.CreateParams {}

export const { useUpdate: useCreate } = Flux.createUpdate<CreateParams>({
  name: RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ data, client }) => {
    const res = await client.arcs.directories.create(data);
    return { ...res, parent: data.parent };
  },
});

export interface RenameParams extends Pick<arc.directory.Directory, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.arcs.directories.rename(data.key, data.name, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export const { useUpdate: useDelete } = Flux.createUpdate<
  arc.directory.Key | arc.directory.Key[]
>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data }) => {
    await client.arcs.directories.delete(data);
    return data;
  },
});

export interface MoveParams {
  source: resource.ID;
  destination: resource.ID;
  ids: resource.ID[];
}

const MOVE_VERBS: verbs.Verbs = {
  present: "move",
  participle: "moving",
  past: "moved",
};

export const { useUpdate: useMove } = Flux.createUpdate<MoveParams>({
  name: RESOURCE_NAME,
  verbs: MOVE_VERBS,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { source, destination, ids } = data;
    await onOptimisticComplete(data);
    await client.arcs.directories.move(source, destination, ...ids);
    return data;
  },
});
