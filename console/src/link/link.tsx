// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch, type UnknownAction } from "@reduxjs/toolkit";
import { type ontology, Synnax, user } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Icon } from "@synnaxlabs/media";
import {
  Menu,
  Status,
  Synnax as PSynnax,
  useAsyncEffect,
  useSyncedRef,
} from "@synnaxlabs/pluto";
import { onOpenUrl } from "@tauri-apps/plugin-deep-link";
import { type ReactElement } from "react";
import { useDispatch, useStore } from "react-redux";

import { Cluster } from "@/cluster";
import { Layout } from "@/layout";
import { Permissions } from "@/permissions";
import { type RootState } from "@/store";

export interface HandlerProps {
  addStatus: (status: Status.CrudeSpec) => void;
  client: Synnax;
  dispatch: Dispatch<UnknownAction>;
  placer: Layout.Placer;
  resource: string;
  resourceKey: string;
  windowKey: string;
}

export type Handler = (props: HandlerProps) => Promise<boolean>;

export interface UseDeepProps {
  handlers: Handler[];
}

const openUrlErrorMessage =
  "Cannot open URL, URLs must be of the form synnax://cluster/<cluster-key> or synnax://cluster/<cluster-key>/<resource>/<resource-key>";
const scheme = "synnax://";

export const useDeep = ({ handlers }: UseDeepProps): void => {
  const client = PSynnax.use();
  const clientRef = useSyncedRef(client);
  const addStatus = Status.useAggregator();
  const dispatch = useDispatch();
  const placer = Layout.usePlacer();
  const store = useStore<RootState>();
  const windowKey = useSelectWindowKey() as string;
  const addOpenUrlErrorStatus = () =>
    addStatus({
      variant: "error",
      message: openUrlErrorMessage,
    });

  useAsyncEffect(async () => {
    const unlisten = await onOpenUrl(async (urls) => {
      dispatch(Drift.focusWindow({}));

      // Processing URL, making sure is has valid form
      if (urls.length === 0 || !urls[0].startsWith(scheme))
        return addOpenUrlErrorStatus();
      const urlParts = urls[0].slice(scheme.length).split("/");
      if ((urlParts.length !== 2 && urlParts.length !== 4) || urlParts[0] !== "cluster")
        return addOpenUrlErrorStatus();

      // Connecting to the cluster
      const clusterKey = urlParts[1];
      const connParams = Cluster.select(store.getState(), clusterKey)?.props;
      const addClusterErrorStatus = () =>
        addStatus({
          variant: "error",
          message: `Cannot open URL, Cluster with key ${clusterKey} not found`,
        });
      if (connParams == null) return addClusterErrorStatus();
      dispatch(Cluster.setActive(clusterKey));
      clientRef.current = new Synnax(connParams);
      if (clientRef.current == null) return addClusterErrorStatus();
      const username = clientRef.current.props.username;
      const user_ = await clientRef.current.user.retrieveByName(username);
      const policies = await clientRef.current.access.policy.retrieveFor(
        user.ontologyID(user_.key),
      );
      dispatch(Permissions.set({ policies }));
      if (urlParts.length === 2) return;

      // Processing the resource part of URL
      const resource = urlParts[2];
      const resourceKey = urlParts[3];
      for (const h of handlers)
        if (
          await h({
            resource,
            resourceKey,
            client: clientRef.current,
            dispatch,
            placer,
            addStatus,
            windowKey,
          })
        )
          return;
      addStatus({
        variant: "error",
        message: `Cannot open link, ${resource} is unknown`,
      });
    });
    return () => unlisten();
  }, []);
};

export const CopyMenuItem = (): ReactElement => (
  <Menu.Item itemKey="link" size="small" startIcon={<Icon.Link />}>
    Copy link
  </Menu.Item>
);

export interface CopyToClipboardProps {
  clusterKey?: string;
  name: string;
  ontologyID?: ontology.IDPayload;
}

export const useCopyToClipboard = (): ((props: CopyToClipboardProps) => void) => {
  const activeClusterKey = Cluster.useSelectActiveKey();
  const addStatus = Status.useAggregator();
  return ({ ontologyID, name, clusterKey }) => {
    let url = "synnax://cluster/";
    const key = clusterKey ?? activeClusterKey;
    if (key == null)
      return addStatus({
        variant: "error",
        message: `Failed to copy link to ${name} to clipboard`,
        description: "No active cluster found",
      });
    url += key;
    if (ontologyID != undefined) url += `/${ontologyID.type}/${ontologyID.key}`;
    navigator.clipboard.writeText(url).then(
      () => {
        addStatus({
          variant: "success",
          message: `Link to ${name} copied to clipboard.`,
        });
      },
      () => {
        addStatus({
          variant: "error",
          message: `Failed to copy link to ${name} to clipboard.`,
        });
      },
    );
  };
};

const urlRegex = new RegExp(
  "^(https?:\\/\\/)?" + // http:// or https:// (optional)
    "((([a-zA-Z0-9][a-zA-Z0-9-]*\\.)+[a-zA-Z]{2,})|" + // domain name and extension
    "localhost|" + // localhost
    "(\\d{1,3}\\.){3}\\d{1,3})" + // or IP address
    "(\\:\\d+)?" + // port (optional)
    "(\\/[-a-zA-Z0-9@:%._\\+~#=]*)*" + // path (optional)
    "(\\?[;&a-zA-Z0-9%_.,~+=-]*)?" + // query string (optional)
    "(#[-a-zA-Z0-9_]*)?$", // fragment identifier (optional)
);

export const isLink = (string: string): boolean => urlRegex.test(string);
