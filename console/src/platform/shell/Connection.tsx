// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type connection } from "@synnaxlabs/client";
import { Status, Synnax, Text } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Island } from "@/platform/shell/Island";

export const STATUS_LABELS: Record<connection.Status["variant"], string> = {
  success: "Connected",
  info: "Connected",
  loading: "Connecting",
  warning: "Reconnecting",
  error: "Unreachable",
  disabled: "Disconnected",
};

export interface ConnectionCluster {
  name: string;
  host: string;
  port: number | string;
  secure?: boolean;
}

export interface ConnectionProps {
  /** Identity of the Core the surface works against; null hides the island. */
  cluster?: ConnectionCluster | null;
}

/**
 * Connection island: the target Core's identity and reachability. The live
 * client status is authoritative when the target is the active connection;
 * otherwise a one-shot check reports reachability.
 */
export const Connection = ({ cluster }: ConnectionProps): ReactElement | null => {
  const client = Synnax.use();
  const live = Synnax.useConnectionStatus();
  const isActive =
    client != null &&
    cluster != null &&
    client.params.host === cluster.host &&
    Number(client.params.port) === Number(cluster.port);
  const checked = Synnax.useCheckConnection(
    isActive || cluster == null
      ? null
      : {
          host: cluster.host,
          port: cluster.port,
          secure: cluster.secure,
          retry: { maxRetries: 0 },
        },
  );
  if (cluster == null) return null;
  const status = isActive ? live : checked;
  const variant = status?.variant ?? "loading";
  return (
    <Island
      gap="medium"
      className={CSS.BE("shell", "connection")}
      data-tauri-drag-region
    >
      <Status.Indicator
        variant={variant}
        className={CSS.BE("shell", "connection-dot")}
      />
      <Text.Text color={10} weight={500} overflow="ellipsis">
        {cluster.name}
      </Text.Text>
      <Text.Text color={9} overflow="ellipsis">
        {cluster.host}:{cluster.port}
      </Text.Text>
      <Text.Text status={variant} className={CSS.BE("shell", "connection-status")}>
        {STATUS_LABELS[variant]}
      </Text.Text>
    </Island>
  );
};
