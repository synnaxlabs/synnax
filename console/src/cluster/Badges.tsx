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
import { caseconv } from "@synnaxlabs/x";
import { type ReactElement } from "react";

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
  state: { status, message },
}: ConnectionStateBadgeProps): ReactElement => (
  <Tooltip.Dialog location={{ x: "left", y: "bottom" }}>
    <Align.Space direction="y" size={0.5}>
      <Status.Text
        variant={statusVariants[status]}
        weight={450}
        hideIcon
        style={{ paddingLeft: 0 }}
      >
        {caseconv.capitalize(status)}
      </Status.Text>
      {message && (
        <Text.Text level="p" color="var(--pluto-gray-l7)" weight={450}>
          {message}
        </Text.Text>
      )}
    </Align.Space>
    <Status.Text
      variant={statusVariants[status]}
      justify="center"
      className={CSS.B("connection-status-badge")}
      style={{
        backgroundColor:
          status === "failed" ? "var(--pluto-error-z-20)" : "var(--pluto-gray-l0)",
        borderColor: "var(--pluto-gray-l4)",
      }}
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
