import type { Connectivity } from "@synnaxlabs/client";
import { Text, Status } from "@synnaxlabs/pluto";
import type { StatusVariant } from "@synnaxlabs/pluto";

import { useSelectActiveCluster } from "../store";
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

/**
 * Displays the name of the active cluster. It must be placed within a react-redux
 * Provider
 */
export const ActiveClusterBadge = (): JSX.Element => {
  const cluster = useSelectActiveCluster();
  return (
    <Text.WithIcon level="p" startIcon={<ClusterIcon />}>
      {cluster != null ? cluster.name : "No Active Cluster"}
    </Text.WithIcon>
  );
};

/**
 * Displays the connection state of the active cluster. It must be placed within a
 * react-redux Provider. If no cluster is active, it displays the default connection
 * state provided by @synnaxlabs/client
 */
export const ActiveConnectionBadge = (): JSX.Element => {
  const cluster = useSelectActiveCluster();
  return <ConnectionStateBadge state={cluster?.state ?? DEFAULT_CONNECTION_STATE} />;
};
