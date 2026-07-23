// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { errors } from "@synnaxlabs/x";

import { clearSelected } from "@/session/project/slice";
import { Synchronizer } from "@/session/synchronizer";

export const SYNCHRONIZERS: Synchronizer.Synchronizers = {
  useClearDeletedProject: Synchronizer.create({
    onDelete: (client, handler) => client.projects.onDelete(handler),
    retrieveExisting: async (client, keys) => {
      const existing: string[] = [];
      for (const key of keys)
        try {
          await client.projects.retrieve(key);
          existing.push(key);
        } catch (err) {
          if (!NotFoundError.matches(err)) throw errors.fromUnknown(err);
        }
      return existing;
    },
    selectKeys: ({ project: { selected } }) => (selected == null ? [] : [selected]),
    remove: () => clearSelected(),
  }),
};
