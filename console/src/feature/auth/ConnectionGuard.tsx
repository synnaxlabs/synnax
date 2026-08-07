// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/auth/Login.css";

import { type connection, type Synnax as Client } from "@synnaxlabs/client";
import { Logo } from "@synnaxlabs/media";
import { Button, Flex, Status, Synnax, Text } from "@synnaxlabs/pluto";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
  useCallback,
  useEffect,
  useState,
} from "react";

import { Login } from "@/feature/auth/Login";
import { LoginNav } from "@/feature/auth/LoginNav";
import { Cluster } from "@/platform/cluster";
import { CSS } from "@/platform/css";
import { Session } from "@/session";

export interface ConnectionGuardProps extends PropsWithChildren {
  /** Renders the window nav chrome above the takeover. Off in child windows. */
  nav?: boolean;
}

/**
 * Blacks out the workspace while the active cluster is unusable or the
 * session is in structural doubt. Rejected credentials return to the login
 * surface at any warmth. Until the session settles a single splash renders
 * instead of the workspace: connecting before first contact, connection
 * trouble with error detail and actions once a probe fails, preparing once
 * the cluster is reached. Warm degradation renders children intact.
 */
export const ConnectionGuard = ({
  children,
  nav = true,
}: ConnectionGuardProps): ReactNode => {
  const client = Synnax.use();
  const status = Synnax.useConnectionStatus();
  const settled = Session.useSettled();
  if (client == null) return children;
  if (status.variant === "error" && status.details.reason === "auth")
    return <Login nav={nav} />;
  if (!settled) return <Splash client={client} status={status} nav={nav} />;
  return children;
};

interface SplashProps {
  client: Client;
  status: connection.Status;
  nav: boolean;
}

const Splash = ({ client, status, nav }: SplashProps): ReactElement => {
  const { variant, details } = status;
  const connecting = details.epoch === 0;
  const troubled =
    connecting &&
    (variant === "error" || details.error != null || details.retry != null);
  return (
    <Flex.Box y empty className={CSS.B("login")}>
      {nav && <LoginNav />}
      <Flex.Box
        y
        align="center"
        justify="center"
        background={1}
        gap="huge"
        grow
        data-tauri-drag-region
        className={CSS.BE("login", "content")}
      >
        <Logo
          variant="title"
          className={CSS.BE("login", "logo")}
          data-tauri-drag-region
        />
        {troubled ? (
          <Trouble client={client} status={status} />
        ) : (
          <Status.Summary
            variant="loading"
            message={connecting ? status.message : "Preparing your workspace..."}
          />
        )}
      </Flex.Box>
    </Flex.Box>
  );
};

interface RetryStatusProps {
  retry: NonNullable<connection.StatusDetails["retry"]>;
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
  const dispatch = Session.useDispatch();
  const logout = Session.useLogout();
  const openConnect = Cluster.useConnectModal();
  const { error, retry } = status.details;

  const handleSelect = useCallback(
    (key?: string) => {
      if (key != null) dispatch(Session.Cluster.select(key));
    },
    [dispatch],
  );

  return (
    <Flex.Box
      pack
      x
      className={CSS.BE("login", "container")}
      grow={false}
      rounded={1.5}
      background={0}
    >
      <Cluster.List
        className={CSS.BE("login", "list")}
        value={activeKey ?? undefined}
        onChange={handleSelect}
      />
      <Flex.Box
        y
        gap="huge"
        className={CSS.BE("login", "form")}
        bordered
        grow
        shrink={false}
        align="center"
        justify="center"
      >
        <Flex.Box y gap="small" align="center">
          <Text.Text level="h2" color={11} weight={450}>
            {cluster?.name ?? "Cluster"}
          </Text.Text>
          {cluster != null && (
            <Text.Text color={9}>
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
        <Flex.Box x gap="small" align="center">
          <Button.Button variant="filled" onClick={() => client.connection.retryNow()}>
            Retry now
          </Button.Button>
          {activeKey != null && (
            <Button.Button
              variant="outlined"
              onClick={() => openConnect({ clusterKey: activeKey })}
            >
              Edit connection
            </Button.Button>
          )}
          <Button.Button variant="outlined" onClick={logout}>
            Log out
          </Button.Button>
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
  );
};
