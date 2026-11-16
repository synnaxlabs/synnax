// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context } from "@synnaxlabs/lyra/context";

const NO_MEMBERS: string[] = [];

const [Context, useContext] = context.create<ReadonlyMap<string, string[]>>({
  defaultValue: new Map(),
  displayName: "GroupBox.Context",
});

/** MembersProvider gives each group box the keys of the nodes it holds. */
export const MembersProvider = Context;

/** useMembers returns the keys of the nodes that the group box holds. */
export const useMembers = (key: string): string[] =>
  useContext().get(key) ?? NO_MEMBERS;
