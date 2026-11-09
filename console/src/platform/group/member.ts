// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, type resource, type Synnax } from "@synnaxlabs/client";

import { type Tree } from "@/platform/tree";

const SCOPES = new Set<resource.Type>(
  Object.values(group.MEMBER_RULES).map(({ scope }) => scope),
);

/** Returns whether a tree holds groups. A tree root names its scope by type. */
export const isScope = (type: resource.Type): boolean => SCOPES.has(type);

/** Lists the records of one type that sit in a group. */
export interface Member {
  type: group.MemberType;
  children: (client: Synnax, group: group.Key) => Tree.Source;
}
