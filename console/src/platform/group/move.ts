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
export const SCOPES = ["channel", "device", "project"] as const;

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

/** Returns where a new group goes when created under the node of a tree. */
export const placement = (
  node: resource.ID,
  root: resource.ID,
): Pick<group.CreateParams, "parent" | "scope"> =>
  resource.idsEqual(node, root) ? { scope: root.type } : { parent: node };
