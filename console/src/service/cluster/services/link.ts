// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type connection, type Synnax as Client } from "@synnaxlabs/client";
import { Synnax, useSyncedRef } from "@synnaxlabs/pluto";
import { breaker, TimeSpan } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch, useStore } from "react-redux";

import { Cluster } from "@/cluster";
import { type Link } from "@/service/link";
import { type RootState } from "@/session/store";

const CONNECT_TIMEOUT = TimeSpan.seconds(10);
const POLL_INTERVAL = TimeSpan.milliseconds(50);
const POLL_BREAKER_CONFIG: breaker.Config = {
  baseInterval: POLL_INTERVAL,
  scale: 1,
  maxRetries: Math.ceil(CONNECT_TIMEOUT.milliseconds / POLL_INTERVAL.milliseconds),
};

// Snapshot is the latest view of the Pluto-managed client and its connection state.
export interface Snapshot {
  client: Client | null;
  connState: connection.State;
}

// ConnectContext supplies connectToCluster with everything it needs to observe and
// mutate connection state without binding to React. Production wires these to the Redux
// store and the Synnax provider; tests inject controllable stubs.
export interface ConnectContext {
  getState: () => RootState;
  getSnapshot: () => Snapshot;
  setActive: (key: string) => void;
  poll: breaker.Breaker;
}

// connectToCluster resolves the cluster identified by key to a connected client. If the
// target cluster is already active and connected, its client is returned immediately.
// Otherwise the active cluster is switched and the function polls the snapshot until the
// provider reports a successful connection to the target, returning that managed client.
// No client is constructed here, so there is nothing to close.
//
// It throws if the cluster is unknown, if the connection fails, or if the connection
// does not succeed before the poll exhausts its retries.
export const connectToCluster = async (
  key: string,
  { getState, getSnapshot, setActive, poll }: ConnectContext,
): Promise<Client> => {
  const state = getState();
  const cluster = Cluster.select(state, key);
  if (cluster == null) throw new Error(`Core with key ${key} not found`);
  const current = getSnapshot();
  if (
    current.client != null &&
    current.connState.status === "connected" &&
    current.connState.clusterKey === key
  )
    return current.client;
  // When switching to a different cluster, the provider tears down the current client
  // and constructs a fresh one. Until that new client exists, connState still describes
  // the previous cluster - including a stale "failed" - so we must not treat a terminal
  // state as belonging to the target yet.
  const switching = Cluster.selectActiveKey(state) !== key;
  const priorClient = current.client;
  setActive(key);
  while (true) {
    const { client, connState } = getSnapshot();
    const attemptStarted = !switching || client !== priorClient;
    if (
      attemptStarted &&
      connState.status === "connected" &&
      connState.clusterKey === key &&
      client != null
    )
      return client;
    if (attemptStarted && connState.status === "failed")
      throw new Error(connState.message ?? `Failed to connect to cluster ${key}`);
    if (!(await poll.wait())) throw new Error(`Timed out connecting to cluster ${key}`);
  }
};

// useLink returns a connect function that resolves a cluster key to a connected client.
// See connectToCluster for the resolution semantics.
export const useLink = (): Link.ClusterConnect => {
  const client = Synnax.use();
  const connState = Synnax.useConnectionState();
  const stateRef = useSyncedRef({ client, connState });
  const dispatch = useDispatch();
  const store = useStore<RootState>();
  return useCallback(
    (key) =>
      connectToCluster(key, {
        getState: () => store.getState(),
        getSnapshot: () => stateRef.current,
        setActive: (key) => dispatch(Cluster.setActive(key)),
        poll: new breaker.Breaker(POLL_BREAKER_CONFIG),
      }),
    [dispatch, store],
  );
};
