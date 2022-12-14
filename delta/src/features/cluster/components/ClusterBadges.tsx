// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Connectivity } from "@synnaxlabs/client";
import { Text, Status } from "@synnaxlabs/pluto";
import type { StatusVariant } from "@synnaxlabs/pluto";

import { useSelectCluster } from "../store";
import { ConnectionState, DEFAULT_CONNECTION_STATE } from "../types";

import { ClusterIcon } from "./ClusterIcon";

/** Props for the ConnectionStateBadge component. */
export interface ConnectionStateBadgeProps {
  state: ConnectionState;
}

const statusVariants: Record<Connectivity, StatusVariant> = {
  connected: "success",
  failed: "error",
  connecting: "info",
  disconnected: "warning",
};

/**
 * A simple badge that displays the connection state of a cluster using an informative
 * text, icon, and color.
 * @param props - The props of the component.
 * @param props.state - The connection state of the cluster.
 */
export const ConnectionStateBadge = ({
  state: { message, status },
}: ConnectionStateBadgeProps): JSX.Element => (
  <Status.Text variant={statusVariants[status]}>{message}</Status.Text>
);

/* The props for the ClusterBadge component. */
export interface ClusterBadgeProps {
  key?: string;
}

/**
 * Displays the name of the cluster.
 *
 * @param props - The props of the component.
 * @param props.key - The key of the cluster to display. If not provided, the active
 * cluster will be used.
 */
export const ClusterBadge = ({ key }: ClusterBadgeProps): JSX.Element => {
  const cluster = useSelectCluster(key);
  return (
    <Text.WithIcon level="p" startIcon={<ClusterIcon />}>
      {cluster?.name ?? "No Active Cluster"}
    </Text.WithIcon>
  );
};

/** The props fo the ConnectionBadge component.  */
type ConnectionBadgeProps = ClusterBadgeProps;

/**
 * Displays the connection state of the cluster.
 *
 * @param props - The props of the component.
 * @param props.key - The key of the cluster to display. If not provided, the active
 * cluster will be used.
 */
export const ConnectionBadge = ({ key }: ConnectionBadgeProps): JSX.Element => {
  const cluster = useSelectCluster(key);
  return <ConnectionStateBadge state={cluster?.state ?? DEFAULT_CONNECTION_STATE} />;
};
