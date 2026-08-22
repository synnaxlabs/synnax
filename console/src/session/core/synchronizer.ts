// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { useRef } from "react";

import { type Action, type StoreState } from "@/session/core/slice";
import { Modals } from "@/session/modals";
import { type Synchronizer } from "@/session/synchronizer";

interface RequiredStoreState extends StoreState, Drift.StoreState {}
type RequiredAction = Action | Drift.Action;

const discard = (
  store: Synchronizer.Store<Drift.StoreState, Drift.Action>,
  modals: Modals.Store,
): void => {
  modals.clear();
  Drift.selectWindows(store.getState())
    .filter(({ key, reserved }) => reserved && key !== Drift.MAIN_WINDOW)
    .forEach(({ key }) => store.dispatch(Drift.closeWindow({ key })));
};

const useCloseOnChange = (): Synchronizer.Callbacks<Drift.StoreState, Drift.Action> => {
  const modals = Modals.useStore("useCloseOnChange");
  const seen = useRef<string | null>(null);
  return {
    reconcile: ({ client, store }) => {
      const { clusterKey } = client.connection.status.details;
      if (clusterKey === "") return;
      const previous = seen.current;
      seen.current = clusterKey;
      // The ref outlives client swaps, so selecting a different Core counts as a
      // change too, not just a replacement at the same address.
      if (previous != null && previous !== clusterKey) discard(store, modals);
    },
  };
};

export const SYNCHRONIZERS: Synchronizer.Synchronizers<
  RequiredStoreState,
  RequiredAction
> = [{ name: "close windows on Core change", use: useCloseOnChange }];
