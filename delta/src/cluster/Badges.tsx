// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import type { connection } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Text, Status, Synnax } from "@synnaxlabs/pluto";
import { Case } from "@synnaxlabs/x";

import { useSelect } from "@/cluster/selectors";

/** Props for the ConnectionStateBadge component. */
export interface ConnectionStateBadgeProps {
  state: connection.State;
}

export const statusVariants: Record<connection.Status, Status.Variant> = {
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
export const ConnectionStatusBadge = ({
  state: { status },
}: ConnectionStateBadgeProps): ReactElement => (
  <Status.Text variant={statusVariants[status]}>{Case.capitalize(status)}</Status.Text>
);

/* The props for the ClusterBadge component. */
export interface NameBadgeProps {
  key?: string;
}

/**
 * Displays the name of the cluster.
 *
 * @param props - The props of the component.
 * @param props.key - The key of the cluster to display. If not provided, the active
 * cluster will be used.
 */
export const NameBadge = ({ key }: NameBadgeProps): ReactElement => {
  const cluster = useSelect(key);
  return (
    <Text.WithIcon level="p" startIcon={<Icon.Cluster />}>
      {cluster?.name ?? "No Active Cluster"}
    </Text.WithIcon>
  );
};

/**
 * Displays the connection state of the cluster.
 *
 * @param props - The props of the component.
 * @param props.key - The key of the cluster to display. If not provided, the active
 * cluster will be used.
 */
export const ConnectionBadge = (): ReactElement => {
  const state = Synnax.useConnectionState();
  return <ConnectionStatusBadge state={state} />;
};
