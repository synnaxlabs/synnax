// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type connection } from "@synnaxlabs/client";
import { Flex, Status, Synnax, Text, Tooltip } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { type ReactElement, useEffect } from "react";
import { useDispatch } from "react-redux";

import { detectConnection } from "@/cluster/detectConnection";
import { Version } from "@/version";

/** Props for the ConnectionStateBadge component. */
export interface ConnectionStateBadgeProps {
  state: connection.State;
}

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
  state: { status, message, clockSkewExcessive, clockSkew },
}: ConnectionStateBadgeProps): ReactElement => {
  let variant = Synnax.CONNECTION_STATE_VARIANTS[status];
  if (status === "connected" && clockSkewExcessive === true) variant = "warning";
  return (
    <Tooltip.Dialog location={location.BOTTOM_LEFT}>
      <Flex.Box y gap="tiny">
        <Text.Text status={variant} weight={650}>
          {STATUS_MESSAGES[status]}
        </Text.Text>
        {message != null && (
          <Text.Text color={9} weight={450}>
            {message}
          </Text.Text>
        )}
        {clockSkewExcessive === true && clockSkew != null && (
          <Text.Text color={9} weight={450}>
            Clock skew: {clockSkew.toString()}
          </Text.Text>
        )}
      </Flex.Box>
      <Text.Text
        status={variant}
        bordered
        borderColor={5}
        background={variant !== "disabled" && 0}
        size="medium"
        rounded
      >
        <Status.Indicator variant={variant} />
      </Text.Text>
    </Tooltip.Dialog>
  );
};

const RemoteVersionUpdater = (): null => {
  const state = Synnax.useConnectionState();
  const dispatch = useDispatch();
  useEffect(() => {
    if (state.status !== "connected") return;
    const version = state.nodeVersion;
    if (version == null) return;
    dispatch(Version.set(version));
  }, [state]);
  return null;
};

export const ConnectionBadge = (): ReactElement => {
  const state = Synnax.useConnectionState();
  const serving = detectConnection();
  return (
    <>
      {serving != null && <RemoteVersionUpdater />}
      <ConnectionStatusBadge state={state} />
    </>
  );
};
