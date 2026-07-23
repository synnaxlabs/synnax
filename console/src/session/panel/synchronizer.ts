// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { purge } from "@/session/panel/slice";
// Type-only: a value import of the store would cycle through its domain barrels.
import type { State } from "@/session/store";
import { Synchronizer } from "@/session/synchronizer";

const selectKeys = (state: State): string[] => {
  const keys = new Set<string>();
  Object.values(state.panels.windows).forEach((win) => {
    if (win.selected != null) keys.add(win.selected);
    Object.keys(win.panels).forEach((key) => keys.add(key));
  });
  return [...keys];
};

export const SYNCHRONIZERS: Synchronizer.Synchronizers = {
  usePruneDeletedPanels: Synchronizer.create({
    onDelete: (client, handler) => client.panels.onDelete(handler),
    retrieveExisting: async (client, keys) =>
      (await client.panels.retrieve({ keys })).map(({ key }) => key),
    selectKeys,
    remove: (keys) => purge(keys),
  }),
};
