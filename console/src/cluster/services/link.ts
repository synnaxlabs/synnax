// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax, useSyncedRef } from "@synnaxlabs/pluto";
import { breaker, TimeSpan } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch, useStore } from "react-redux";

import { Cluster } from "@/cluster";
import { type Link } from "@/layered/service/link";
import { type RootState } from "@/store";

const CONNECT_TIMEOUT = TimeSpan.seconds(10);
const POLL_INTERVAL = TimeSpan.milliseconds(50);
const POLL_BREAKER_CONFIG: breaker.Config = {
  baseInterval: POLL_INTERVAL,
  scale: 1,
  maxRetries: Math.ceil(CONNECT_TIMEOUT.milliseconds / POLL_INTERVAL.milliseconds),
};

// useLink returns a connect function that resolves a cluster key to a connected client.
// If the cluster is already active and connected, the managed client is returned
// immediately. Otherwise it switches the active cluster and waits for the Pluto-managed
// provider to reconnect to the target, returning that managed client. No client is
// constructed here, so there is nothing to close.
export const useLink = (): Link.ClusterConnect => {
  const client = Synnax.use();
  const connState = Synnax.useConnectionState();
  const stateRef = useSyncedRef({ client, connState });
  const dispatch = useDispatch();
  const store = useStore<RootState>();
  return useCallback(
    async (key) => {
      const state = store.getState();
      const cluster = Cluster.select(state, key);
      if (cluster == null) throw new Error(`Core with key ${key} not found`);
      const current = stateRef.current;
      if (
        current.client != null &&
        current.connState.status === "connected" &&
        current.connState.clusterKey === key
      )
        return current.client;
      // When switching to a different cluster, the provider tears down the current
      // client and constructs a fresh one. Until that new client exists, connState
      // still describes the previous cluster - including a stale "failed" - so we must
      // not treat a terminal state as belonging to the target yet.
      const switching = Cluster.selectActiveKey(state) !== key;
      const priorClient = current.client;
      dispatch(Cluster.setActive(key));
      const poll = new breaker.Breaker(POLL_BREAKER_CONFIG);
      while (true) {
        const { client, connState } = stateRef.current;
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
        if (!(await poll.wait()))
          throw new Error(`Timed out connecting to cluster ${key}`);
      }
    },
    [dispatch, store],
  );
};
