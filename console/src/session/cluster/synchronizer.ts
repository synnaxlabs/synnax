// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type connection } from "@synnaxlabs/client";

import { type Action, changeKey, type StoreState } from "@/session/cluster/slice";
import { type Synchronizer } from "@/session/synchronizer";

const repair = (
  store: Synchronizer.Store<StoreState, Action>,
  status: connection.Status,
): void => {
  if (status.variant !== "success") return;
  const { clusterKey } = status.details;
  const selected = store.getState().cluster.selected;
  if (selected == null || clusterKey === "" || selected === clusterKey) return;
  store.dispatch(changeKey({ oldKey: selected, newKey: clusterKey }));
};

// Adopts the cluster key the connection reports: connecting to a different
// cluster at the same address, or the predefined local/demo clusters.
const syncKey: Synchronizer.Callbacks<StoreState, Action> = {
  reconcile: ({ client, store }) => repair(store, client.connection.status),
  listen: ({ client, store }) =>
    client.connection.onChange((status) => repair(store, status)),
};

export const SYNCHRONIZERS: Synchronizer.Synchronizers<StoreState, Action> = [
  { name: "sync cluster key", use: () => syncKey },
];
