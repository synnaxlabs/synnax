// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/auth/ConnectionGuard.css";

import { type connection } from "@synnaxlabs/client";
import { Button, Flex, Icon, Status, Synnax, Text } from "@synnaxlabs/pluto";
import {
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
  useEffect,
  useState,
} from "react";

import { Login } from "@/feature/auth/Login";
import { Shell } from "@/feature/shell";
import { Cluster } from "@/platform/cluster";
import { Connection } from "@/platform/connection";
import { CSS } from "@/platform/css";
import { Shell as PlatformShell } from "@/platform/shell";
import { Session } from "@/session";

/**
 * Renders a splash instead of the workspace until the session settles. Rejected
 * credentials return to the login surface; a degraded live connection does not.
 */
export const ConnectionGuard = ({ children }: PropsWithChildren): ReactNode => {
  const client = Synnax.use();
  const status = Synnax.useConnectionStatus();
  const settled = Session.useSettled();
  if (client == null) return children;
  if (status.variant === "error" && status.details.reason === "auth") return <Login />;
  if (!settled) return <Splash status={status} />;
  return children;
};

interface CountdownCoreProps {
  retry: NonNullable<connection.StatusDetails["retry"]>;
  checking: boolean;
}

const CountdownCore = ({ retry, checking }: CountdownCoreProps): ReactElement => {
  const remaining = Connection.useCountdown(retry.nextAt);
  return (
    <>
      <Text.Text level="h3" color={11} className={CSS.BE("connection", "countdown")}>
        {checking ? <Icon.Loading /> : `${remaining}s`}
      </Text.Text>
      <Text.Text level="small" color={9}>
        <Icon.Sync />
        {retry.attempt}
      </Text.Text>
    </>
  );
};

interface SplashProps {
  status: connection.Status;
}

const Splash = ({ status }: SplashProps): ReactElement => {
  const { variant, details } = status;
  const activeKey = Session.Cluster.useSelectSelectedKey();
  const cluster = Session.Cluster.useSelectState(activeKey ?? undefined);
  const connecting = details.epoch === 0;
  const troubled =
    connecting &&
    (variant === "error" || details.error != null || details.retry != null);
  // Fast connections settle before this fires, so the card never flashes a spinner.
  const [revealed, setRevealed] = useState(false);
  useEffect(() => {
    const timeout = setTimeout(() => setRevealed(true), 300);
    return () => clearTimeout(timeout);
  }, []);
  const checking = Connection.useHeldChecking(details.checking);
  const core =
    troubled && details.retry != null ? (
      <CountdownCore retry={details.retry} checking={checking} />
    ) : undefined;
  return (
    // Trouble puts the connection detail in the card, so the island would repeat it.
    <Shell.Frame className={CSS.B("connection")} connection={troubled ? null : cluster}>
      <Flex.Box
        y
        align="center"
        justify="center"
        gap={8}
        className={CSS(CSS.BE("connection", "body"), revealed && CSS.M("revealed"))}
      >
        <Status.Orbital core={<PlatformShell.Mark>{core}</PlatformShell.Mark>} />
        {troubled ? (
          <Trouble />
        ) : (
          <Status.Summary
            variant="loading"
            message={connecting ? status.message : "Preparing your workspace..."}
          />
        )}
      </Flex.Box>
    </Shell.Frame>
  );
};

const Trouble = (): ReactElement => {
  const activeKey = Session.Cluster.useSelectSelectedKey();
  const logout = Session.useLogout();
  const openConnect = Cluster.useConnectModal();
  return (
    <Flex.Box y gap="large" full="x">
      <Connection.Target />
      <Flex.Box y gap="small" full="x">
        <Connection.Retry variant="filled" size="large" full="x" justify="center" />
        <Flex.Box x gap="small" full="x" className={CSS.BE("connection", "actions")}>
          {activeKey != null && (
            <Button.Button
              variant="outlined"
              grow
              justify="center"
              onClick={() => openConnect({ clusterKey: activeKey })}
            >
              <Icon.Edit />
              Edit connection
            </Button.Button>
          )}
          <Button.Button variant="outlined" grow justify="center" onClick={logout}>
            <Icon.Logout />
            Log out
          </Button.Button>
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
  );
};
