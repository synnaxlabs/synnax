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
import { Logo } from "@synnaxlabs/media";
import { Button, Flex, Status, Synnax, Text } from "@synnaxlabs/pluto";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
  useEffect,
  useState,
} from "react";

import { Login } from "@/feature/auth/Login";
import { Cluster } from "@/platform/cluster";
import { CSS } from "@/platform/css";
import { Shell } from "@/platform/shell";
import { Session } from "@/session";

/**
 * Blacks out the workspace while the active cluster is unusable or the
 * session is in structural doubt. Rejected credentials return to the login
 * surface at any warmth. Until the session settles a single splash renders
 * instead of the workspace: connecting before first contact, connection
 * trouble with error detail and actions once a probe fails, preparing once
 * the cluster is reached. Warm degradation renders children intact.
 */
export const ConnectionGuard = ({ children }: PropsWithChildren): ReactNode => {
  const client = Synnax.use();
  const status = Synnax.useConnectionStatus();
  const settled = Session.Settled.use();
  if (client == null) return children;
  if (status.variant === "error" && status.details.reason === "auth") return <Login />;
  if (!settled) return <Splash client={client} status={status} />;
  return children;
};

interface SplashProps {
  client: Client;
  status: connection.Status;
}

const Splash = ({ client, status }: SplashProps): ReactElement => {
  const { variant, details } = status;
  const connecting = details.epoch === 0;
  const troubled =
    connecting &&
    (variant === "error" || details.error != null || details.retry != null);
  // Fast connections settle before the reveal timer fires, so the splash stays
  // an empty card instead of flashing a spinner for a few frames.
  const [revealed, setRevealed] = useState(false);
  useEffect(() => {
    const timeout = setTimeout(() => setRevealed(true), 300);
    return () => clearTimeout(timeout);
  }, []);
  return (
    <Shell.Frame className={CSS.B("connection")}>
      <Flex.Box
        y
        align="center"
        justify="center"
        gap="huge"
        className={CSS(CSS.BE("connection", "body"), revealed && CSS.M("revealed"))}
      >
        <Flex.Box center grow={false} className={CSS.BE("shell", "mark-ring")}>
          <Logo variant="icon" className={CSS.BE("shell", "mark")} />
        </Flex.Box>
        {troubled ? (
          <Trouble client={client} status={status} />
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

interface RetryStatusProps {
  retry: NonNullable<connection.Details["retry"]>;
}

const RetryStatus = ({ retry }: RetryStatusProps): ReactElement => {
  const [now, setNow] = useState(() => TimeStamp.now());
  useEffect(() => {
    const interval = setInterval(() => setNow(TimeStamp.now()), 500);
    return () => clearInterval(interval);
  }, []);
  const remaining = Math.max(
    0,
    Math.ceil(new TimeSpan(retry.nextAt.valueOf() - now.valueOf()).seconds),
  );
  return (
    <Text.Text color={9}>
      Attempt {retry.attempt} failed. Retrying in {remaining}s
    </Text.Text>
  );
};

interface TroubleProps {
  client: Client;
  status: connection.Status;
}

const Trouble = ({ client, status }: TroubleProps): ReactElement => {
  const activeKey = Session.Cluster.useSelectSelectedKey();
  const cluster = Session.Cluster.useSelectState(activeKey ?? undefined);
  const logout = Session.useLogout();
  const openConnect = Cluster.useConnectModal();
  const { error, retry } = status.details;
  return (
    <>
      <Flex.Box
        y
        align="center"
        gap="tiny"
        grow={false}
        className={CSS.BE("connection", "target")}
      >
        <Text.Text color={10} weight={500}>
          {cluster?.name ?? "Cluster"}
        </Text.Text>
        {cluster != null && (
          <Text.Text level="small" color={9}>
            {cluster.host}:{cluster.port}
          </Text.Text>
        )}
      </Flex.Box>
      <Flex.Box y gap="small" align="center">
        <Status.Summary variant={status.variant} message={status.message} />
        {error != null && error.message !== status.message && (
          <Text.Text color={9}>{error.message}</Text.Text>
        )}
        {retry != null && <RetryStatus retry={retry} />}
      </Flex.Box>
      <Flex.Box y gap="small" full="x">
        <Button.Button
          variant="filled"
          size="large"
          full="x"
          justify="center"
          onClick={() => client.connection.retryNow()}
        >
          Retry Now
        </Button.Button>
        <Flex.Box x gap="small" full="x">
          {activeKey != null && (
            <Button.Button
              variant="outlined"
              grow
              justify="center"
              onClick={() => openConnect({ clusterKey: activeKey })}
            >
              Edit Connection
            </Button.Button>
          )}
          <Button.Button variant="outlined" grow justify="center" onClick={logout}>
            Log Out
          </Button.Button>
        </Flex.Box>
      </Flex.Box>
    </>
  );
};
