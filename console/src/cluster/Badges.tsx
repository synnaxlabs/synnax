// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/cluster/Badges.css";

import { type connection } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";

import { CSS } from "@/css";

/** Props for the ConnectionStateBadge component. */
export interface ConnectionStateBadgeProps {
  state: connection.State;
}

export const statusVariants: Record<connection.Status, Status.Variant> = {
  connected: "success",
  failed: "error",
  connecting: "loading",
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
}: ConnectionStateBadgeProps) => (
  <Status.Text
    className={CSS.B("connection-status-badge")}
    variant={statusVariants[status]}
    justify="center"
  >
    {caseconv.capitalize(status)}
  </Status.Text>
);

/**
 * Displays the connection state of the cluster.
 */
export const ConnectionBadge = () => {
  const state = Synnax.useConnectionState();
  return <ConnectionStatusBadge state={state} />;
};
