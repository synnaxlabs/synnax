// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/auth/ConnectionGuard.css";

import { type connection, type Synnax as Client } from "@synnaxlabs/client";
import { Button, Flex, Icon, Status, Synnax, Text } from "@synnaxlabs/pluto";
import { TimeSpan } from "@synnaxlabs/x";
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
  if (!settled) return <Splash client={client} status={status} />;
  return children;
};

// A check against a dead local port fails in milliseconds, too fast to see.
const CHECK_HOLD = TimeSpan.milliseconds(1250);

const useHeldChecking = (checking: boolean): boolean => {
  const [held, setHeld] = useState(checking);
  useEffect(() => {
    if (checking) {
      setHeld(true);
      return;
    }
    const timeout = setTimeout(() => setHeld(false), CHECK_HOLD.milliseconds);
    return () => clearTimeout(timeout);
  }, [checking]);
  return held;
};

interface CountdownCoreProps {
  retry: NonNullable<connection.StatusDetails["retry"]>;
  checking: boolean;
}

const CountdownCore = ({ retry, checking }: CountdownCoreProps): ReactElement => {
  const remaining = PlatformShell.useCountdown(retry.nextAt);
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
  client: Client;
  status: connection.Status;
}

const Splash = ({ client, status }: SplashProps): ReactElement => {
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
  const checking = useHeldChecking(details.checking);
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
          <Trouble client={client} status={status} checking={checking} />
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

interface TroubleProps {
  client: Client;
  status: connection.Status;
  checking: boolean;
}

const Trouble = ({ client, status, checking }: TroubleProps): ReactElement => {
  const activeKey = Session.Cluster.useSelectSelectedKey();
  const cluster = Session.Cluster.useSelectState(activeKey ?? undefined);
  const logout = Session.useLogout();
  const openConnect = Cluster.useConnectModal();
  const { variant } = status;
  return (
    <Flex.Box y gap="large" full="x">
      <Flex.Box
        x
        align="center"
        justify="center"
        gap="medium"
        className={CSS.BE("connection", "target")}
      >
        <Status.Indicator variant={variant} />
        <Text.Text color={10} weight={500} overflow="ellipsis">
          {cluster?.name ?? "Cluster"}
        </Text.Text>
        {cluster != null && (
          <Text.Text color={9} overflow="ellipsis">
            {cluster.host}:{cluster.port}
          </Text.Text>
        )}
        <Text.Text status={variant} className={CSS.BE("connection", "status")}>
          <span>{checking ? "Retrying" : PlatformShell.STATUS_LABELS[variant]}</span>
          <span className={CSS.M("ghost")} aria-hidden>
            Retrying
          </span>
          <span className={CSS.M("ghost")} aria-hidden>
            {PlatformShell.STATUS_LABELS[variant]}
          </span>
        </Text.Text>
      </Flex.Box>
      <Flex.Box y gap="small" full="x">
        <Button.Button
          variant="filled"
          size="large"
          full="x"
          justify="center"
          disabled={checking}
          onClick={() => client.connection.retryNow()}
        >
          <Icon.Refresh />
          Retry now
        </Button.Button>
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
