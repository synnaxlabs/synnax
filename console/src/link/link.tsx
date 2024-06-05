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
} from "@synnaxlabs/pluto";
import { ReactElement } from "react";
import { onOpenUrl, getCurrent } from "@tauri-apps/plugin-deep-link";
import { Dispatch, UnknownAction } from "@reduxjs/toolkit";
import { useDispatch, useStore } from "react-redux";

import { Cluster } from "@/cluster";
import { Layout } from "@/layout";

export interface HandlerProps {
  // url is a string of two parts, the first part is the resource identifier and
  // the second part is the resource key.
  resource: string;
  resourceKey: string;
  client: Synnax;
  dispatch: Dispatch<UnknownAction>;
  placer: Layout.Placer;
  clusters: Cluster.Cluster[];
}

// export type Handler = (props: HandlerProps) => Promise<void>;
export type Handler = (props: HandlerProps) => Promise<boolean>;

export interface UseDeepLinkProps {
  handlers: Handler[];
}

export const useDeep = ({ handlers }: UseDeepLinkProps): void => {
  console.log("useDeep");
  const client = PSynnax.use();
  if (client == null) return;
  const clientRef = useSyncedRef(client);

  const dispatch = useDispatch();
  const placer = Layout.usePlacer();

  const clusters = Cluster.useSelectMany();

  const store = useStore();
  // store.
  const currentCluster = Cluster.useSelectActiveKey();
  console.log("Current cluster ", currentCluster);
  console.log("Client is at", client?.props.port);

  // TODO: add drift window focusing
  useAsyncEffect(async () => {
    console.log("useAsyncEffect");
    const currUrl = getCurrent();
    console.log("Current URL is", currUrl);
    const unlisten = await onOpenUrl(async (urls) => {
      console.log("onOpenUrl");
      console.log(`URL is ${urls[0]}`);
      // drift window focusing here
      // dispatch(Drift.focusWindow());
      const scheme = "synnax://";
      if (urls.length === 0 || !urls[0].startsWith(scheme)) {
        console.error("Error: Cannot open URL, URLs must start with synnax://");
        return;
      }
      const urlParts = urls[0].slice(scheme.length).split("/");

      if (urlParts.length !== 2 && urlParts.length !== 4) {
        console.error(
          "Error: Cannot open URL, URLs must be of the form synnax://cluster/<cluster-key> or synnax://cluster/<cluster-key>/<resource>/<resource-key>",
        );
        return;
      }
      if (urlParts[0] !== "cluster") {
        console.log("Invalid URL");
        return;
      }
      const clusterKey = urlParts[1];

      const stork = store.getState();
      // const connParams = Cluster.select(store.getState(), clusterKey)?.props;
      // if (connParams == null) return;
      // const client = new Synnax(connParams);

      console.log("Opened cluster, Trying to find handlers");
      for (let h of handlers)
        if (
          await h({
            resource: urlParts[2],
            resourceKey: urlParts[3],
            client,
            dispatch,
            placer,
            clusters,
          })
        ) {
          console.log("Handler found that returns true");
          break;
        } else {
          console.log("This handler returned false");
        }
    });
    console.log("Return out of the effect");
    return () => unlisten();
  }, []);
};

export const CopyMenuItem = (): ReactElement => (
  <Menu.Item itemKey="link" startIcon={<Icon.Link />}>
    Copy link address
  </Menu.Item>
);

// TODO: 2) asynch function 3) focus drift window 4)
// Notifications 5) Add cluster link to cluster tab 6) Add range link to range
// tab 7) Add workspace link to workspace tab
