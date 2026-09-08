// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/core/Badge.css";

import { type connection, status as clientStatus } from "@synnaxlabs/client";
import {
  Button,
  Dialog,
  Divider,
  Flex,
  Icon,
  Synnax,
  Text,
  Tooltip,
} from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Clipboard } from "@/platform/clipboard";
import { Connection } from "@/platform/connection";
import { Core } from "@/platform/core";
import { CSS } from "@/platform/css";
import { User } from "@/platform/user";
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

const Content = (): ReactElement => {
  const status = Synnax.useConnectionStatus();
  const { variant, details } = status;
  const activeKey = Session.Core.useSelectSelectedKey();
  const core = Session.Core.useSelectSelected();
  const copy = Clipboard.useCopy();
  const openConnect = Core.useConnectModal();
  const handleLogout = Session.useLogout();
  const { close } = Dialog.useContext();
  const degraded =
    variant === "loading" || variant === "warning" || variant === "error";
  const skewDirection = details.clockSkew.valueOf() > 0n ? "ahead of" : "behind";
  const copyAddress = (): void => {
    if (core != null) copy(`${core.host}:${core.port}`, "Core address");
  };
  const editConnection = (): void => {
    if (activeKey == null) return;
    close();
    openConnect({ coreKey: activeKey });
  };
  return (
    <>
      <Flex.Box y gap="small" className={CSS.BE("core-badge", "body")}>
        <Flex.Box
          x
          align="center"
          gap="medium"
          className={CSS.BE("core-badge", "name")}
        >
          <Text.Text weight={500} color={10} overflow="ellipsis">
            {core?.name ?? "Core"}
          </Text.Text>
          {activeKey != null && (
            <Button.Button
              aria-label="Edit connection"
              variant="text"
              size="tiny"
              onClick={editConnection}
            >
              <Icon.Edit />
            </Button.Button>
          )}
          <Text.Text
            status={variant}
            level="small"
            className={CSS.BE("core-badge", "state")}
          >
            {Connection.STATUS_LABELS[variant]}
          </Text.Text>
        </Flex.Box>
        {core != null && (
          <Flex.Box
            x
            align="center"
            gap="small"
            className={CSS.BE("core-badge", "address")}
          >
            <Text.Text
              level="small"
              color={9}
              overflow="ellipsis"
              onClick={copyAddress}
            >
              {core.host}:{core.port}
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
      {activeKey != null && (
        <>
          <Divider.Divider x />
          <User.Info />
        </>
      )}
      {(degraded || activeKey != null) && (
        <Flex.Box x gap="small" className={CSS.BE("core-badge", "actions")}>
          {degraded && (
            <Connection.Retry variant="filled" size="small" grow justify="center" />
          )}
          {activeKey != null && (
            <Button.Button
              onClick={handleLogout}
              variant="filled"
              status="error"
              size="small"
              grow
              justify="center"
            >
              <Icon.Logout />
              Log out
            </Button.Button>
          )}
        </Flex.Box>
      )}
    </>
  );
};

export const Badge = (): ReactElement => {
  const status = Synnax.useConnectionStatus();
  const { variant } = status;
  const name = User.useDisplayName();
  return (
    <Dialog.Frame>
      <Tooltip.Dialog location={location.BOTTOM_LEFT}>
        <Summary status={status} />
        {/* Button disables itself on "loading" and "disabled", and the badge must
            stay clickable while disconnected. */}
        <Dialog.Trigger
          aria-label="Core menu"
          hideCaret
          variant="outlined"
          size="medium"
          rounded="small"
          gap="small"
          status={clientStatus.removeVariants(variant, ["loading", "disabled"])}
        >
          <Text.Text className={CSS.BE("core-badge", "user")}>
            <Icon.User />
            {name}
          </Text.Text>
          <Connection.Indicator />
        </Dialog.Trigger>
      </Tooltip.Dialog>
      <Dialog.Dialog
        bordered
        borderColor={7}
        className={CSS.BE("core-badge", "dialog")}
      >
        <Content />
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
