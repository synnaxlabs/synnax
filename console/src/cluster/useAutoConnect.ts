// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect } from "react";
import { useDispatch } from "react-redux";

import { detectServingCluster } from "@/cluster/autoConnect";
import { useSelectMany, useSelectActiveKey } from "@/cluster/selectors";
import { set, setActive } from "@/cluster/slice";
import { Modals } from "@/modals";

/**
 * Hook that automatically connects to the serving cluster if the console
 * is being hosted by a Synnax cluster and no other cluster is active.
 */
export const useAutoConnect = (): void => {
  const dispatch = useDispatch();
  const clusters = useSelectMany(); // Get all clusters
  const activeClusterKey = useSelectActiveKey();
  const promptCredentials = Modals.useCredentials();

  useEffect(() => {
    // Only attempt auto-connect if there's no active cluster
    if (activeClusterKey != null) return;

    // Detect the serving cluster
    const servingCluster = detectServingCluster();
    if (servingCluster == null) return;

    // // Check if this cluster already exists in the store
    // const existingCluster = clusters.find(
    //   (cluster) =>
    //     cluster.host === servingCluster.host && cluster.port === servingCluster.port,
    // );

    // if (existingCluster != null) {
    //   // Cluster exists, just set it as active
    //   dispatch(setActive(existingCluster.key));
    //   return;
    // }

    // Need to prompt for credentials and create a new cluster
    const connectWithCredentials = async () => {
      console.log("connectWithCredentials");
      const credentials = await promptCredentials({
        host: `${servingCluster.host}:${servingCluster.port}`,
      });

      if (credentials == null) return; // User cancelled

      const clusterWithCredentials = {
        ...servingCluster,
        name: `Auto-connected (${servingCluster.host})`,
        username: credentials.username,
        password: credentials.password,
      };

      dispatch(set(clusterWithCredentials));
      dispatch(setActive(clusterWithCredentials.key));
    };

    void connectWithCredentials();
  }, [dispatch, clusters, activeClusterKey, promptCredentials]);
};
