// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, type status } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/pluto";
import { destructor, errors } from "@synnaxlabs/x";
import { useMemo } from "react";

import { type Action, removeFavorites, type StoreState } from "@/session/status/slice";
import { type Synchronizer } from "@/session/synchronizer";

// Reports whether the status is gone, by the only answer that proves it: the Core
// saying so. Absence from a listing does not, since a listing can be older than the
// favorite it is compared against.
const isDeleted = async (
  client: Synchronizer.Params<StoreState, Action>["client"],
  key: status.Key,
): Promise<boolean> => {
  try {
    await client.statuses.retrieve(key);
    return false;
  } catch (err) {
    if (NotFoundError.matches(err)) return true;
    throw errors.fromUnknown(err);
  }
};

const useSyncStatuses: Synchronizer.Use<StoreState, Action> = () => {
  const addStatus = Status.useAdder();
  return useMemo(
    () => ({
      reconcile: async ({ client, store }) => {
        const favorites = store.getState().status.favorites;
        if (favorites.length === 0) return;
        const deleted = await Promise.all(
          favorites.map(async (key) => ((await isDeleted(client, key)) ? key : null)),
        );
        const gone = deleted.filter((key) => key != null);
        if (gone.length > 0) store.dispatch(removeFavorites(gone));
      },
      listen: ({ client, store }) => {
        const removeOnSet = client.statuses.onSet(addStatus);
        const removeOnDelete = client.statuses.onDelete((key) =>
          store.dispatch(removeFavorites(key)),
        );
        return () => destructor.unwind(removeOnSet, removeOnDelete);
      },
    }),
    [addStatus],
  );
};

export const SYNCHRONIZERS: Synchronizer.Synchronizers<StoreState, Action> = [
  { name: "sync statuses", use: useSyncStatuses },
];
