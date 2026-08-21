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

import { type Action, removeFavorites, type StoreState } from "@/session/status/slice";
import { Synchronizer } from "@/session/synchronizer";

// Statuses the cluster raises while connected surface as notifications. Nothing is
// repaired at a boundary: a notification missed during a gap has no state behind it.
const useForwardStatuses: Synchronizer.Use<StoreState, Action> = () => {
  const addStatus = Status.useAdder();
  return useMemo(
    () => ({ listen: ({ client }) => client.statuses.onSet(addStatus) }),
    [addStatus],
  );
};

export const SYNCHRONIZERS: Synchronizer.Synchronizers<StoreState, Action> = [
  Synchronizer.createRemover<StoreState, Action>({
    name: "remove deleted status favorites",
    domain: (client) => client.statuses,
    selectKeys: (state: StoreState) => state.status.favorites,
    remove: (keys) => removeFavorites(keys),
  }),
  { name: "forward statuses to notifications", use: useForwardStatuses },
];
