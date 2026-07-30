// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type connection } from "@synnaxlabs/client";
import { Flex, Status, Synnax, Text } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Cluster } from "@/platform/cluster";
import { CSS } from "@/platform/css";

// Rotation switches: "chip" floats a pill above the bottom edge, "bar" pins a
// full-width strip to it; placement positions the chip along that edge.
type Variant = "chip" | "bar";
const VARIANT = "chip" as Variant;

type Placement = "center" | "left" | "right";
const PLACEMENT = "center" as Placement;

export const STATUS_LABELS: Record<connection.Status["variant"], string> = {
  success: "Connected",
  info: "Connected",
  loading: "Connecting",
  warning: "Reconnecting",
  error: "Unreachable",
  disabled: "Disconnected",
};

export interface FooterCluster {
  name: string;
  host: string;
  port: number | string;
  secure?: boolean;
}

export interface FooterProps {
  /** Identity of the Core the surface works against; null hides the footer. */
  cluster?: FooterCluster | null;
}

/**
 * Connection footer: the target Core's identity and reachability. The live
 * client status is authoritative when the target is the active connection;
 * otherwise a one-shot probe reports reachability.
 */
export const Footer = ({ cluster }: FooterProps): ReactElement | null => {
  const client = Synnax.use();
  const live = Synnax.useConnectionStatus();
  const isActive =
    client != null &&
    cluster != null &&
    client.params.host === cluster.host &&
    Number(client.params.port) === Number(cluster.port);
  const probed = Cluster.useReachability(isActive ? null : cluster);
  if (cluster == null) return null;
  const status = isActive ? live : probed;
  const variant = status?.variant ?? "loading";
  return (
    <Flex.Box
      x
      align="center"
      gap="medium"
      className={CSS(
        CSS.BE("shell", "footer"),
        CSS.M(`footer-${VARIANT}`),
        CSS.M(`footer-${PLACEMENT}`),
      )}
    >
      <Status.Indicator variant={variant} className={CSS.BE("shell", "footer-dot")} />
      <Text.Text color={10} weight={500} overflow="ellipsis">
        {cluster.name}
      </Text.Text>
      <Text.Text color={9} overflow="ellipsis">
        {cluster.host}:{cluster.port}
      </Text.Text>
      <Text.Text status={variant} className={CSS.BE("shell", "footer-status")}>
        {STATUS_LABELS[variant]}
      </Text.Text>
    </Flex.Box>
  );
};
