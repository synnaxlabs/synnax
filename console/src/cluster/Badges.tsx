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
import { Align, Status, Synnax, Text, Tooltip } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/css";

/** Props for the ConnectionStateBadge component. */
export interface ConnectionStateBadgeProps {
  state: connection.State;
}

export const STATUS_VARIANTS: Record<connection.Status, Status.Variant> = {
  connected: "success",
  failed: "error",
  connecting: "loading",
  disconnected: "info",
};

const STATUS_MESSAGES: Record<connection.Status, string> = {
  connected: "Connected",
  failed: "Error",
  connecting: "Connecting",
  disconnected: "Disconnected",
};

/**
 * A simple badge that displays the connection state of a cluster using an informative
 * text, icon, and color.
 * @param props - The props of the component.
 * @param props.state - The connection state of the cluster.
 */
export const ConnectionStatusBadge = ({
  state: { status, message },
}: ConnectionStateBadgeProps): ReactElement => (
  <Tooltip.Dialog location={{ x: "left", y: "bottom" }}>
    <Align.Space y size="tiny">
      <Status.Text
        variant={STATUS_VARIANTS[status]}
        weight={650}
        hideIcon
        style={{ paddingLeft: 0 }}
      >
        {STATUS_MESSAGES[status]}
      </Status.Text>
      {message != null && (
        <Text.Text level="p" shade={9} weight={450}>
          {message}
        </Text.Text>
      )}
    </Align.Space>
    <Status.Text
      variant={STATUS_VARIANTS[status]}
      justify="center"
      className={CSS(CSS.B("connection-status-badge"), CSS.M(status))}
    />
  </Tooltip.Dialog>
);

/**
 * Displays the connection state of the cluster.
 */
export const ConnectionBadge = (): ReactElement => {
  const state = Synnax.useConnectionState();
  return <ConnectionStatusBadge state={state} />;
};
