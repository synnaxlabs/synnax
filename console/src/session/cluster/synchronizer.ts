// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type connection } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { useRef } from "react";

import { type Action, changeKey, type StoreState } from "@/session/cluster/slice";
import { Modals } from "@/session/modals";
import { type Synchronizer } from "@/session/synchronizer";

interface RequiredStoreState extends StoreState, Drift.StoreState {}
type RequiredAction = Action | Drift.Action;

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
const syncKey: Synchronizer.Synchronizer<StoreState, Action> = {
  reconcile: ({ client, store }) => repair(store, client.connection.status),
  listen: ({ client, store }) =>
    client.connection.onChange((status) => repair(store, status)),
};

const discard = (
  store: Synchronizer.Store<Drift.StoreState, Drift.Action>,
  modals: Modals.Store,
): void => {
  modals.clear();
  Drift.selectWindows(store.getState())
    .filter(({ key, reserved }) => reserved && key !== Drift.MAIN_WINDOW)
    .forEach(({ key }) => store.dispatch(Drift.closeWindow({ key })));
};

// Torn-off windows and open modals hold resources from the cluster that is no
// longer connected. Main window only: a closed window takes its modals with it.
const useCloseOnClusterChange = (): Synchronizer.Synchronizer<
  Drift.StoreState,
  Drift.Action
> => {
  const modals = Modals.useStore("useCloseOnClusterChange");
  const seen = useRef<string | null>(null);
  return {
    reconcile: ({ client, store }) => {
      const { clusterKey } = client.connection.status.details;
      if (clusterKey === "") return;
      const previous = seen.current;
      seen.current = clusterKey;
      // The ref outlives client swaps, so selecting a different cluster counts
      // as a change too, not just a replacement at the same address.
      if (previous != null && previous !== clusterKey) discard(store, modals);
    },
  };
};

export const SYNCHRONIZERS: Synchronizer.Synchronizers<
  RequiredStoreState,
  RequiredAction
> = {
  useSyncClusterKey: () => syncKey,
  useCloseOnClusterChange,
};
