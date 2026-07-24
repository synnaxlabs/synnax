// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { remove } from "@/session/log/slice";
import { Synchronizer } from "@/session/synchronizer";

export const SYNCHRONIZERS: Synchronizer.Synchronizers = {
  usePruneDeletedLogs: Synchronizer.create({
    onDelete: (client, handler) => client.logs.onDelete(handler),
    retrieveExisting: async (client, keys) =>
      (await client.logs.retrieve({ keys, ignoreNotFoundError: true })).map(
        ({ key }) => key,
      ),
    selectKeys: (state) => Object.keys(state.log.logs),
    remove: (keys) => remove({ keys }),
  }),
};
