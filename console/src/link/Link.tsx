// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Layout } from "@/layout";
import { Synnax } from "@synnaxlabs/pluto";
import { useEffect } from "react";
import { useDispatch } from "react-redux";
import { setActive } from "@/workspace/slice";
import { onOpenUrl } from "@tauri-apps/plugin-deep-link";
import { Cluster } from "@/cluster";

export const useDeepLink = () => {
  const client = Synnax.use();
  const d = useDispatch();
  const activeClusterKey = Cluster.useSelectActiveKey();
  const clusters = Cluster.useSelectMany();

  useEffect(() => {
    onOpenUrl((urls) => {
      const scheme = "synnax://";
      if (urls.length === 0 || !urls[0].startsWith(scheme)) return;
      const path = urls[0].slice(scheme.length);
      const url = path.split('/');

      // processing the cluster connection
      if (url[0] !== "cluster") {
        console.error("Error: invalid URL");
        return;
      }
      const getCluster = clusters.filter((cluster) => cluster.key === url[1]);
      if (getCluster.length == 0) {
        // Console does not have this cluster in store
        // TODO: logic
        // return (prompt connect a cluster?)
        return;
      }
      d(Cluster.setActive(url[1]));

      // process the workspace
      if (url[2] === 'workspace') {
        const promise = client?.workspaces.retrieve(url[3]);
        if (promise == undefined) return;
        promise
          .then((ws) => {
            if (ws == null) return;
            d(
              Layout.setWorkspace({
                slice: ws.layout as unknown as Layout.SliceState,
              }),
            );
            d(setActive(ws.key));
          })
          .catch((error) => {
            console.error("Error: ", error);
            return;
          });
      } else {
        return;
      }
    });
  }, [client]);
};
