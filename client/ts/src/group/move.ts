// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type batch } from "@/batch";
import { type Client } from "@/group/client";
import { type MoveRequest } from "@/group/member";
import { MEMBER_RULES, type MemberType } from "@/group/members.gen";
import { type Key } from "@/group/types.gen";
import { resource } from "@/resource";

interface Mover<K extends number | string> {
  setGroupRequest: MoveRequest<K>;
}

/** The clients that move records into groups. A Synnax client is one. */
export interface Movers {
  groups: Pick<Client, "setParentRequest">;
  channels: Mover<number>;
  racks: Mover<number>;
  devices: Mover<string>;
  projects: Mover<string>;
  schematics: Mover<string> & { symbols: Mover<string> };
  lineplots: Mover<string>;
  logs: Mover<string>;
  tables: Mover<string>;
  panels: Mover<string>;
}

const SET_GROUP: Record<
  MemberType,
  (movers: Movers, keys: string[], to?: Key) => batch.Request
> = {
  channel: (m, keys, to) => m.channels.setGroupRequest(keys.map(Number), to),
  device: (m, keys, to) => m.devices.setGroupRequest(keys, to),
  lineplot: (m, keys, to) => m.lineplots.setGroupRequest(keys, to),
  log: (m, keys, to) => m.logs.setGroupRequest(keys, to),
  panel: (m, keys, to) => m.panels.setGroupRequest(keys, to),
  project: (m, keys, to) => m.projects.setGroupRequest(keys, to),
  rack: (m, keys, to) => m.racks.setGroupRequest(keys.map(Number), to),
  schematic: (m, keys, to) => m.schematics.setGroupRequest(keys, to),
  schematic_symbol: (m, keys, to) => m.schematics.symbols.setGroupRequest(keys, to),
  table: (m, keys, to) => m.tables.setGroupRequest(keys, to),
};

/** Returns whether a record of the type can sit in a group. */
export const isMember = (type: resource.Type): type is MemberType =>
  type in MEMBER_RULES;

/**
 * Builds the batch requests that move resources under a parent: a group, project, or
 * rack, or the top of their tree when the parent is undefined. A group takes the
 * parent as its parent. Any other resource joins the parent when it is a group, and
 * leaves its group when it is not.
 * @throws if a resource is of a type that cannot sit in a group.
 */
export const moveRequests = (
  movers: Movers,
  ids: resource.ID[],
  parent?: resource.ID,
): batch.Request[] => {
  const byType = new Map<resource.Type, string[]>();
  ids.forEach(({ type, key }) => byType.set(type, [...(byType.get(type) ?? []), key]));
  const to = parent?.type === "group" ? parent.key : undefined;
  return [...byType].map(([type, keys]) => {
    if (type === "group") return movers.groups.setParentRequest(keys, parent);
    if (!isMember(type)) throw new Error(`a ${type} cannot sit in a group`);
    return SET_GROUP[type](movers, keys, to);
  });
};

/** A resource to move. */
export interface Moved {
  id: resource.ID;
  /** The ancestors of the resource, nearest first, ending with the root of its tree. */
  ancestors: resource.ID[];
}

export interface CanMoveParams {
  /** The root of the tree. */
  root: resource.ID;
  /** The resource to move under. */
  destination: resource.ID;
  /** The ancestors of the destination, nearest first, ending with the root. */
  ancestors: resource.ID[];
  /** The resources to move. */
  moved: Moved[];
}

const rackOf = (ids: resource.ID[]): resource.ID | undefined =>
  ids.find(({ type }) => type === "rack");

/**
 * Returns whether moveRequests can move the resources under the destination. A resource
 * under a rack stays under that rack, and a device under another device stays there.
 * Any other pinned type goes only into a group with no rack or project above it.
 */
export const canMove = ({
  root,
  destination,
  ancestors,
  moved,
}: CanMoveParams): boolean => {
  const above = [destination, ...ancestors];
  const atTop = above.every((id) => id.type === "group" || resource.idsEqual(id, root));
  const rack = rackOf(above);
  return moved.every(({ id, ancestors }) => {
    // A device is the one pinned type with a holder: its parent device or its rack.
    if (id.type === "device") {
      if (ancestors[0]?.type === "device") return false;
    } else if (isMember(id.type) && MEMBER_RULES[id.type].pinned) return atTop;
    const from = rackOf(ancestors);
    return from == null || rack == null ? from === rack : resource.idsEqual(from, rack);
  });
};
