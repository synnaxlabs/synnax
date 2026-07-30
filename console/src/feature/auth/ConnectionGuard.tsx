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

/** How long a cold connect may stay in loading before the takeover mounts. */
const GRACE = TimeSpan.milliseconds(500);

/** True once active has been continuously true for the given delay. */
const useDelayedTrue = (active: boolean, delay: TimeSpan): boolean => {
  const [delayed, setDelayed] = useState(false);
  useEffect(() => {
    if (!active) {
      setDelayed(false);
      return;
    }
    const timeout = setTimeout(() => setDelayed(true), delay.milliseconds);
    return () => clearTimeout(timeout);
  }, [active, delay.milliseconds]);
  return delayed;
};

/**
 * Blacks out the workspace while the active cluster is unusable. Rejected
 * credentials return to the login surface at any warmth; a cluster that has
 * never been reached this session (cold) shows a connecting takeover while
 * coming up or retrying. A fast connect never flashes the takeover: cold
 * loading renders children until it has persisted past a grace period. Warm
 * unreachable degradation renders children intact.
 */
export const ConnectionGuard = ({
  children,
  nav = true,
}: ConnectionGuardProps): ReactNode => {
  const client = Synnax.use();
  const status = Synnax.useConnectionStatus();
  const coldLoading = status.details.epoch === 0 && status.variant === "loading";
  const graceElapsed = useDelayedTrue(coldLoading, GRACE);
  if (client == null) return children;
  if (status.variant === "error" && status.details.reason === "auth")
    return <Login nav={nav} />;
  const coldUnreachable =
    status.details.epoch === 0 &&
    status.variant === "error" &&
    status.details.reason === "unreachable";
  if (coldUnreachable || (coldLoading && graceElapsed))
    return <Takeover client={client} status={status} nav={nav} />;
  return children;
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

interface TakeoverProps {
  client: Client;
  status: connection.Status;
  nav: boolean;
}

const Takeover = ({ client, status, nav }: TakeoverProps): ReactElement => {
  const activeKey = Session.Cluster.useSelectSelectedKey();
  const cluster = Session.Cluster.useSelectState(activeKey ?? undefined);
  const dispatch = Session.useDispatch();
  const logout = Session.useLogout();
  const openConnect = Cluster.useConnectModal();

  const handleSelect = useCallback(
    (key?: string) => {
      if (key != null) dispatch(Session.Cluster.select(key));
    },
    [dispatch],
  );

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
              {status.details.retry != null && (
                <RetryStatus retry={status.details.retry} />
              )}
            </Flex.Box>
            <Flex.Box x gap="small" align="center">
              <Button.Button
                variant="filled"
                onClick={() => client.connection.retryNow()}
              >
                Retry Now
              </Button.Button>
              {activeKey != null && (
                <Button.Button
                  variant="outlined"
                  onClick={() => openConnect({ clusterKey: activeKey })}
                >
                  Edit Connection
                </Button.Button>
              )}
              <Button.Button variant="outlined" onClick={logout}>
                Log Out
              </Button.Button>
            </Flex.Box>
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
  );
};
