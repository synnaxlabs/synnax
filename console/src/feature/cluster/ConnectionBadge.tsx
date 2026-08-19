// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/cluster/ConnectionBadge.css";

import { type connection, status as clientStatus } from "@synnaxlabs/client";
import { Button, Dialog, Flex, Icon, Synnax, Text, Tooltip } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Clipboard } from "@/platform/clipboard";
import { Cluster } from "@/platform/cluster";
import { Connection } from "@/platform/connection";
import { CSS } from "@/platform/css";
import { Session } from "@/session";

interface SummaryProps {
  status: connection.Status;
}

const Summary = ({ status: { variant, message } }: SummaryProps): ReactElement => (
  <Flex.Box y gap="tiny">
    <Text.Text status={variant} weight={650}>
      {Connection.STATUS_LABELS[variant]}
    </Text.Text>
    <Text.Text color={9} weight={450}>
      {message}
    </Text.Text>
  </Flex.Box>
);

const Diagnostics = (): ReactElement => {
  const status = Synnax.useConnectionStatus();
  const { variant, details } = status;
  const activeKey = Session.Cluster.useSelectSelectedKey();
  const cluster = Session.Cluster.useSelectState(activeKey ?? undefined);
  const copy = Clipboard.useCopy();
  const openConnect = Cluster.useConnectModal();
  const { close } = Dialog.useContext();
  const degraded =
    variant === "loading" || variant === "warning" || variant === "error";
  const skewDirection = details.clockSkew.valueOf() > 0n ? "ahead of" : "behind";
  const copyAddress = (): void => {
    if (cluster != null) copy(`${cluster.host}:${cluster.port}`, "cluster address");
  };
  return (
    <>
      <Flex.Box y gap="small" className={CSS.BE("connection-badge", "body")}>
        <Flex.Box x align="center" gap="medium">
          <Connection.Indicator className={CSS.BE("connection-badge", "dot")} />
          <Text.Text weight={500} color={10} overflow="ellipsis">
            {cluster?.name ?? "Core"}
          </Text.Text>
          <Text.Text
            status={variant}
            level="small"
            className={CSS.BE("connection-badge", "state")}
          >
            {Connection.STATUS_LABELS[variant]}
          </Text.Text>
        </Flex.Box>
        {cluster != null && (
          <Flex.Box
            x
            align="center"
            gap="small"
            className={CSS.BE("connection-badge", "address")}
          >
            <Text.Text
              level="small"
              color={9}
              overflow="ellipsis"
              onClick={copyAddress}
            >
              {cluster.host}:{cluster.port}
            </Text.Text>
            <Button.Button variant="text" size="tiny" onClick={copyAddress}>
              <Icon.Copy />
            </Button.Button>
          </Flex.Box>
        )}
        {degraded && details.error != null && (
          <Text.Text level="small" status={variant === "error" ? "error" : "warning"}>
            {details.error.message}
          </Text.Text>
        )}
        {degraded && <Connection.RetrySchedule details={details} />}
        {details.nodeVersion != null && (
          <Text.Text level="small" color={9}>
            {`Core v${details.nodeVersion}`}
          </Text.Text>
        )}
        {details.nodeVersion != null && !details.clientServerCompatible && (
          <Text.Text level="small" status="warning">
            {`Incompatible with client v${details.clientVersion}`}
          </Text.Text>
        )}
        {details.clockSkewExceeded && (
          <Text.Text level="small" status="warning">
            {`Clock is ${details.clockSkew.abs().toString()} ${skewDirection} the Core`}
          </Text.Text>
        )}
      </Flex.Box>
      {(degraded || activeKey != null) && (
        <Flex.Box x gap="small" className={CSS.BE("connection-badge", "actions")}>
          {degraded && (
            <Connection.Retry variant="filled" size="small" grow justify="center" />
          )}
          {activeKey != null && (
            <Button.Button
              variant="outlined"
              size="small"
              grow
              justify="center"
              onClick={() => {
                close();
                openConnect({ clusterKey: activeKey });
              }}
            >
              <Icon.Edit />
              Edit connection
            </Button.Button>
          )}
        </Flex.Box>
      )}
    </>
  );
};

export const ConnectionBadge = (): ReactElement => {
  const status = Synnax.useConnectionStatus();
  const { variant } = status;
  return (
    <Dialog.Frame>
      <Tooltip.Dialog location={location.BOTTOM_LEFT}>
        <Summary status={status} />
        {/* Button disables itself on "loading" and "disabled", and the badge must
            stay clickable while disconnected. */}
        <Dialog.Trigger
          aria-label="Connection status"
          hideCaret
          variant="outlined"
          size="medium"
          rounded="small"
          status={clientStatus.removeVariants(variant, ["loading", "disabled"])}
        >
          <Connection.Indicator />
        </Dialog.Trigger>
      </Tooltip.Dialog>
      <Dialog.Dialog
        bordered
        borderColor={7}
        className={CSS.BE("connection-badge", "dialog")}
      >
        <Diagnostics />
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
