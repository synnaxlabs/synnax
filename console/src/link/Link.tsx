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
import { Workspace } from "@/workspace";
import { onOpenUrl } from "@tauri-apps/plugin-deep-link";
import { Cluster } from "@/cluster";
import { Range } from "@/range";
import { Schematic } from "@/schematic";

export const useDeepLink = (): void => {
  const client = Synnax.use();
  const dispatch = useDispatch();
  const clusters = Cluster.useSelectMany();

  useEffect(() => {
    onOpenUrl((urls) => {
      const scheme = "synnax://";
      if (urls.length === 0 || !urls[0].startsWith(scheme)) {
        // TODO: Do something instead of logging an error?
        console.error("Error: Cannot open URL, URLs must start with synnax://.");
        return;
      }

      const path = urls[0].slice(scheme.length);
      const urlParts = path.split("/");

      // processing the cluster connection
      if (urlParts[0] !== "cluster") {
        // TODO: Do something instead of logging an error?
        console.error("Error: Cannot open URL, no cluster specified.");
        return;
      }
      const clusterKey = urlParts[1];
      const getCluster = clusters.filter((cluster) => cluster.key === clusterKey);
      if (getCluster.length == 0) {
        // Console does not have this cluster in store.
        // TODO: better logic up above instead of using filtering?
        // TODO: open window to connect a cluster, then reload URL?
        console.error(
          `Error: Cannot open URL, cluster with key ${clusterKey} is not found.`,
        );
        return;
      }
      dispatch(Cluster.setActive(clusterKey));
      if (urlParts.length === 2) {
        // only a cluster URL was used.
        return;
      }

      const resource = urlParts[2];
      const resourceKey = urlParts[3];
      // TODO: figure out way to compress below code into a function so there is
      // less repetition.
      console.log("About to use a resource");
      switch (resource) {
        case "workspace":
          const workspacePromise = client?.workspaces.retrieve(resourceKey);
          if (workspacePromise == undefined) return; // TODO: log error here?
          workspacePromise
            .then((workspace) => {
              if (workspace == null) return; // TODO: log error here?
              dispatch(
                Layout.setWorkspace({
                  slice: workspace.layout as unknown as Layout.SliceState,
                }),
              );
              dispatch(Workspace.setActive(workspace.key));
              return;
            })
            .catch((error) => {
              // TODO: figure out difference between null ws, undefined promise,
              // and error.
              console.error("Error: ", error);
              return;
            });
        case "range":
          const rangePromise = client?.ranges.retrieve(resourceKey);
          if (rangePromise == undefined) return; // TODO: log error here?
          rangePromise
            .then((range) => {
              if (range == null) return; // TODO: log error here?
              dispatch(Range.setActive(range.key));
              // TODO: some type of popup here after we set the active range
              return;
            })
            .catch((error) => {
              // TODO: different error here?
              console.error("Error: ", error);
              return;
            });
        case "schematic":
          console.log("Using a schematic");
          const schematicPromise = client?.workspaces.schematic.retrieve(resourceKey);
          if (schematicPromise == undefined) return; // TODO: log error here?
          console.log("Schematic promise is not undefined")
          schematicPromise
            .then((schematic) => {
              if (schematic == null) return; // TODO: log error here?
              console.log("Schematic is not null");
              // TODO: do something
              // const newSchematic = Schematic.create({
              //   ...(schematic.data as unknown as Schematic.State),
              //   key: schematic.key,
              //   name: schematic.name,
              //   snapshot: schematic.snapshot,
              // });
              dispatch(Layout.setWorkspace({ slice: schematic.data as unknown as Layout.SliceState }));
              return;
            })
            .catch((error) => {
              // TODO: different error here?
              console.error("Error: ", error);
              return;
            });
          
          return;
        case "lineplot":
          const linePlotPromise = client?.workspaces.linePlot.retrieve(resourceKey);
          if (linePlotPromise == undefined) return; // TODO: log error here?
          linePlotPromise
            .then((linePlot) => {
              if (linePlot == null) return; // log: add error here?
              // TODO: Do something

              return;
            })
            .catch((error) => {
              // TODO: different error here?
              console.error("Error: ", error);
              return;
            });
          return;
        case "channel":
          const channelPromise = client?.channels.retrieve(resourceKey);
          if (channelPromise == undefined) return; // TODO: log error here?
          channelPromise
            .then((channelPromise) => {
              if (channelPromise == null) return; // log error here?
              // TODO: Do something
              return;
            })
            .catch((error) => {
              // TODO: different error here?
              console.error("Error: ", error);
              return;
            });
          return;
        case "device":
          const devicePromise = client?.hardware.devices.retrieve(resourceKey);
          if (devicePromise == undefined) return; // TODO: log error here?
          devicePromise
            .then((devicePromise) => {
              if (devicePromise == null) return; // log error here?
              // TODO: Do something
              return;
            })
            .catch((error) => {
              // TODO: different error here?
              console.error("Error: ", error);
              return;
            });
        case "task":
          const taskPromise = client?.hardware.tasks.retrieve(resourceKey);
          if (taskPromise == undefined) return; // TODO: log error here?
          taskPromise
            .then((taskPromise) => {
              if (taskPromise == null) return; // log error here?
              // TODO: Do something
              return;
            })
            .catch((error) => {
              // TODO: different error here?
              console.error("Error: ", error);
              return;
            });
        default:
          // TODO: change this?
          console.error("Error: ", `Resource ${resource} could not be found.`);
          return;
      }
    });
  }, [client]);
};
