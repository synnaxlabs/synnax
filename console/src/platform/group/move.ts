// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type batch, type group, resource, type Synnax } from "@synnaxlabs/client";

import { type Tree } from "@/platform/tree";

/** The scopes of the trees that hold groups. A tree root names its scope by type. */
const SCOPES = ["channel", "device", "project"] as const;

export type Scope = (typeof SCOPES)[number];

export const isScope = (type: resource.Type): type is Scope =>
  (SCOPES as readonly resource.Type[]).includes(type);

/** Lists the records of one type that sit in a group. */
export interface Member {
  type: resource.Type;
  children: (client: Synnax, group: group.Key) => Tree.Source;
}

/** The member types each scope holds. */
export type Members = Record<Scope, Member[]>;

const SET_GROUP: Partial<
  Record<
    resource.Type,
    (client: Synnax, keys: string[], to?: group.Key) => batch.Request
  >
> = {
  channel: (client, keys, to) => client.channels.setGroupRequest(keys.map(Number), to),
  device: (client, keys, to) => client.devices.setGroupRequest(keys, to),
  rack: (client, keys, to) => client.racks.setGroupRequest(keys.map(Number), to),
  project: (client, keys, to) => client.projects.setGroupRequest(keys, to),
  schematic: (client, keys, to) => client.schematics.setGroupRequest(keys, to),
  lineplot: (client, keys, to) => client.lineplots.setGroupRequest(keys, to),
  log: (client, keys, to) => client.logs.setGroupRequest(keys, to),
  table: (client, keys, to) => client.tables.setGroupRequest(keys, to),
  panel: (client, keys, to) => client.panels.setGroupRequest(keys, to),
};

/**
 * Builds the batch requests that move resources under a parent: a group, project, or
 * rack, or the top level of the tree when the parent is undefined. A group takes the
 * parent as its parent. Any other resource joins the parent when it is a group, and
 * leaves its group when it is not.
 * @throws if a resource is of a type that cannot sit in a group.
 */
export const moveRequests = (
  client: Synnax,
  ids: resource.ID[],
  parent?: resource.ID,
): batch.Request[] => {
  const byType = new Map<resource.Type, string[]>();
  ids.forEach(({ type, key }) => byType.set(type, [...(byType.get(type) ?? []), key]));
  const to = parent?.type === "group" ? parent.key : undefined;
  return [...byType].map(([type, keys]) => {
    if (type === "group") return client.groups.setParentRequest(keys, parent);
    const setGroup = SET_GROUP[type];
    if (setGroup == null) throw new Error(`a ${type} cannot sit in a group`);
    return setGroup(client, keys, to);
  });
};

const rackOf = (ids: resource.ID[]): resource.ID | undefined =>
  ids.find(({ type }) => type === "rack");

/**
 * Returns whether moveRequests can move the dropped items under the destination. An
 * item under a rack stays under that rack, and a device under another device stays
 * there. A rack or project goes only into a group with no rack or project above it.
 */
export const canMove = ({
  root,
  destination,
  ancestors,
  moved,
}: Pick<
  Tree.CanDropProps,
  "root" | "destination" | "ancestors" | "moved"
>): boolean => {
  const above = [destination.id, ...ancestors];
  const atRoot = above.every(
    (id) => id.type === "group" || resource.idsEqual(id, root),
  );
  const rack = rackOf(above);
  return moved.every(({ id, ancestors }) => {
    if (id.type === "rack" || id.type === "project") return atRoot;
    if (id.type === "device" && ancestors[0]?.type === "device") return false;
    const from = rackOf(ancestors);
    return from == null || rack == null ? from === rack : resource.idsEqual(from, rack);
  });
};
