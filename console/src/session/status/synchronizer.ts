// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/pluto";
import { useMemo } from "react";

import {
  type Action,
  filterFavoritesToKeys,
  removeFavorites,
} from "@/session/status/slice";
import { type Synchronizer } from "@/session/synchronizer";

const useSyncStatuses: Synchronizer.Use<unknown, Action> = () => {
  const addStatus = Status.useAdder();
  return useMemo(
    () => ({
      reconcile: async ({ client, store }) => {
        const statuses = await client.statuses.retrieve({});
        store.dispatch(filterFavoritesToKeys(statuses.map(({ key }) => key)));
      },
      listen: ({ client, store }) => {
        const removeOnSet = client.statuses.onSet(addStatus);
        const removeOnDelete = client.statuses.onDelete((key) =>
          store.dispatch(removeFavorites(key)),
        );
        return () => {
          removeOnSet();
          removeOnDelete();
        };
      },
    }),
    [addStatus],
  );
};

export const SYNCHRONIZERS: Synchronizer.Synchronizers = { useSyncStatuses };
