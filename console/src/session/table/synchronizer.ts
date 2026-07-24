// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synchronizer } from "@/session/synchronizer";
import { remove } from "@/session/table/slice";

export const SYNCHRONIZERS: Synchronizer.Synchronizers = {
  usePruneDeletedTables: Synchronizer.create({
    onDelete: (client, handler) => client.tables.onDelete(handler),
    retrieveExisting: async (client, keys) =>
      (await client.tables.retrieve({ keys, ignoreNotFoundError: true })).map(
        ({ key }) => key,
      ),
    selectKeys: (state) => Object.keys(state.table.tables),
    remove: (keys) => remove({ keys }),
  }),
};
