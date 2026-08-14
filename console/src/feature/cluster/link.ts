// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { Synnax, useSyncedRef } from "@synnaxlabs/pluto";
import { breaker, TimeSpan } from "@synnaxlabs/x";
import { useCallback } from "react";

import { type Link } from "@/platform/link";
import { Session } from "@/session";
import { type State } from "@/session/store";

const CONNECT_TIMEOUT = TimeSpan.seconds(10);
const SWAP_POLL_INTERVAL = TimeSpan.milliseconds(50);
const SWAP_POLL_BREAKER_CONFIG: breaker.Config = {
  baseInterval: SWAP_POLL_INTERVAL,
  scale: 1,
  maxRetries: Math.ceil(CONNECT_TIMEOUT.milliseconds / SWAP_POLL_INTERVAL.milliseconds),
};

// ConnectContext supplies connectToCluster with everything it needs to observe and
// mutate connection state without binding to React. Production wires these to the Redux
// store and the Synnax provider; tests inject controllable stubs.
export interface ConnectContext {
  getState: () => State;
  getClient: () => Client | null;
  setActive: (key: string) => void;
  poll: breaker.Breaker;
}

// connectToCluster resolves the cluster identified by key to a connected client. If the
// target cluster is already active, its managed client's connect() is awaited and the
// client returned. Otherwise the active cluster is switched, the provider-constructed
// replacement client is awaited (client identity changes only on a param change), and
// its connect() drives the outcome. No client is constructed here, so there is nothing
// to close.
//
// It throws if the cluster is unknown, if the provider never swaps clients before the
// poll exhausts its retries, or with the client's typed rejection when the connection
// fails.
export const connectToCluster = async (
  key: string,
  { getState, getClient, setActive, poll }: ConnectContext,
): Promise<Client> => {
  const state = getState();
  const cluster = Session.Cluster.selectState(state, key);
  if (cluster == null) throw new Error(`Core with key ${key} not found`);
  const prior = getClient();
  if (Session.Cluster.selectSelectedKey(state) === key && prior != null) {
    await prior.connect({ timeout: CONNECT_TIMEOUT });
    return prior;
  }
  setActive(key);
  while (true) {
    const client = getClient();
    if (client != null && client !== prior) {
      await client.connect({ timeout: CONNECT_TIMEOUT });
      return client;
    }
    if (!(await poll.wait())) throw new Error(`Timed out connecting to cluster ${key}`);
  }
};

// useLink returns a connect function that resolves a cluster key to a connected client.
// See connectToCluster for the resolution semantics.
export const useLink = (): Link.ClusterConnect => {
  const client = Synnax.use();
  const clientRef = useSyncedRef(client);
  const dispatch = Session.useDispatch();
  const store = Session.useStore();
  return useCallback(
    (key) =>
      connectToCluster(key, {
        getState: () => store.getState(),
        getClient: () => clientRef.current,
        setActive: (key) => dispatch(Session.Cluster.select(key)),
        poll: new breaker.Breaker(SWAP_POLL_BREAKER_CONFIG),
      }),
    [dispatch, store],
  );
};
