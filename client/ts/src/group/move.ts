// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MEMBER_RULES, type MemberType } from "@/group/members.gen";
import { resource } from "@/resource";

/** Returns whether a record of the type can sit in a group. */
export const isMember = (type: resource.Type): type is MemberType =>
  type in MEMBER_RULES;

/** A resource to move. */
export interface Moved {
  id: resource.ID;
  /** The ancestors of the resource, nearest first, ending with the root of its tree. */
  ancestors: resource.ID[];
}

/** The move that {@link canMove} checks. */
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
 * Returns whether {@link Client.moveRequests} can move the resources under the
 * destination. A resource under a rack stays under that rack, and a device under
 * another device stays there. Any other pinned type goes only into a group with no rack
 * or project above it.
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
