// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax, useSyncedRef } from "@synnaxlabs/pluto";
import { sleep, TimeSpan } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch, useStore } from "react-redux";

import { Cluster } from "@/cluster";
import { type Link } from "@/layered/service/link";
import { type RootState } from "@/store";

const CONNECT_TIMEOUT = TimeSpan.seconds(10);
const POLL_INTERVAL = TimeSpan.milliseconds(50);

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
      const cluster = Cluster.select(store.getState(), key);
      if (cluster == null) throw new Error(`Core with key ${key} not found`);
      const current = stateRef.current;
      if (
        current.client != null &&
        current.connState.status === "connected" &&
        current.connState.clusterKey === key
      )
        return current.client;
      dispatch(Cluster.setActive(key));
      const deadline = Date.now() + CONNECT_TIMEOUT.milliseconds;
      for (;;) {
        await sleep.sleep(POLL_INTERVAL);
        const { client, connState } = stateRef.current;
        if (
          connState.status === "connected" &&
          connState.clusterKey === key &&
          client != null
        )
          return client;
        if (connState.status === "failed")
          throw new Error(connState.message ?? `Failed to connect to cluster ${key}`);
        if (Date.now() > deadline)
          throw new Error(`Timed out connecting to cluster ${key}`);
      }
    },
    [dispatch, store],
  );
};
