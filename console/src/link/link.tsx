// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Synnax as PSynnax,
  Menu,
  useAsyncEffect,
  useSyncedRef,
  Status,
} from "@synnaxlabs/pluto";
import { ReactElement } from "react";
import { onOpenUrl } from "@tauri-apps/plugin-deep-link";
import { Dispatch, UnknownAction } from "@reduxjs/toolkit";
import { useDispatch, useStore } from "react-redux";
import { Drift } from "@synnaxlabs/drift";

import { Cluster } from "@/cluster";
import { Layout } from "@/layout";

export interface HandlerProps {
  resource: string;
  resourceKey: string;
  client: Synnax;
  dispatch: Dispatch<UnknownAction>;
  placer: Layout.Placer;
  addStatus: (status: Status.CrudeSpec) => void;
}

export type Handler = (props: HandlerProps) => Promise<boolean>;

export interface UseDeepLinkProps {
  handlers: Handler[];
}

export const useDeep = ({ handlers }: UseDeepLinkProps): void => {
  const client = PSynnax.use();
  const clientRef = useSyncedRef(client);
  const addStatus = Status.useAggregator();
  const dispatch = useDispatch();
  const placer = Layout.usePlacer();
  const store = useStore();
  const openUrlErrorMessage =
    "Error: Cannot open URL, URLs must be of the form synnax://cluster/<cluster-key> or synnax://cluster/<cluster-key>/<resource>/<resource-key>";
  const addOpenUrlErrorStatus = () => {
    addStatus({
      variant: "error",
      key: "openUrlError",
      message: openUrlErrorMessage,
    });
  };

  useAsyncEffect(async () => {
    const unlisten = await onOpenUrl(async (urls) => {
      dispatch(Drift.focusWindow({}));

      // Processing URL, making sure is has valid form
      const scheme = "synnax://";
      if (urls.length === 0 || !urls[0].startsWith(scheme)) {
        addOpenUrlErrorStatus();
        return;
      }
      const urlParts = urls[0].slice(scheme.length).split("/");
      if (urlParts.length !== 2 && urlParts.length !== 4) {
        addOpenUrlErrorStatus();
        return;
      }
      if (urlParts[0] !== "cluster") {
        addOpenUrlErrorStatus();
        return;
      }

      // Connecting to the cluster
      const clusterKey = urlParts[1];
      const connParams = Cluster.select(
        store.getState() as Cluster.StoreState,
        clusterKey,
      )?.props;
      const addClusterErrorStatus = () => {
        addStatus({
          variant: "error",
          key: "openUrlError-${clusterKey}",
          message: `Error: Cannot open URL, Cluster with key ${clusterKey} not found`,
        });
      };
      if (connParams == null) {
        addClusterErrorStatus();
        return;
      }
      dispatch(Cluster.setActive(clusterKey));
      clientRef.current = new Synnax(connParams);
      if (clientRef.current == null) {
        addClusterErrorStatus();
        return;
      }
      if (urlParts.length === 2) return;

      // Processing the resource part of URL
      const resource = urlParts[2];
      const resourceKey = urlParts[3];
      for (let h of handlers)
        if (
          await h({
            resource,
            resourceKey,
            client: clientRef.current,
            dispatch,
            placer,
            addStatus,
          })
        )
          return;
      addStatus({
        variant: "error",
        key: "openUrlError-ResourceNotFound-",
        message: `Error: Cannot open URL, ${resource} is not a valid resource type`,
      });
    });
    return () => unlisten();
  }, []);
};

export const CopyMenuItem = (): ReactElement => (
  <Menu.Item itemKey="link" size="small" startIcon={<Icon.Link />}>
    Copy resource URL
  </Menu.Item>
);
