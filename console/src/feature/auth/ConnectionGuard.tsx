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

import { CredentialsForm } from "@/feature/auth/CredentialsForm";
import { LoginNav } from "@/feature/auth/LoginNav";
import { Cluster } from "@/platform/cluster";
import { CSS } from "@/platform/css";
import { Session } from "@/session";

export interface ConnectionGuardProps extends PropsWithChildren {
  /** Renders the window nav chrome above the takeover. Off in child windows. */
  nav?: boolean;
}

const takesOver = ({ variant, reason, epoch }: connection.State): boolean => {
  if (variant === "error" && reason === "auth") return true;
  return (
    epoch === 0 &&
    (variant === "loading" || (variant === "error" && reason === "unreachable"))
  );
};

/**
 * Blacks out the workspace with a full-screen connection surface while the
 * active cluster has never been reached this session (cold) and the connection
 * is still coming up or has failed, or whenever credentials are rejected.
 * Warm unreachable degradation renders children intact.
 */
export const ConnectionGuard = ({
  children,
  nav = true,
}: ConnectionGuardProps): ReactNode => {
  const client = Synnax.use();
  const state = Synnax.useConnectionState();
  if (client == null || !takesOver(state)) return children;
  return <Takeover client={client} state={state} nav={nav} />;
};

interface RetryStatusProps {
  retry: NonNullable<connection.State["retry"]>;
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
  state: connection.State;
  nav: boolean;
}

const Takeover = ({ client, state, nav }: TakeoverProps): ReactElement => {
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

  const isAuth = state.variant === "error" && state.reason === "auth";

  let body: ReactElement;
  if (isAuth)
    body = (
      <Flex.Box
        pack
        x
        className={CSS(CSS.BE("login", "container"), CSS.M("narrow"))}
        grow={false}
        rounded={1.5}
        background={0}
      >
        <Flex.Box
          y
          gap="huge"
          className={CSS.BE("login", "form")}
          bordered
          grow
          shrink={false}
        >
          <Status.Summary variant="error" message={state.message} />
          <CredentialsForm />
        </Flex.Box>
      </Flex.Box>
    );
  else
    body = (
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
            <Status.Summary variant={state.variant} message={state.message} />
            {state.retry != null && <RetryStatus retry={state.retry} />}
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
        {body}
        {isAuth && (
          <Button.Button variant="text" onClick={logout}>
            Log Out
          </Button.Button>
        )}
      </Flex.Box>
    </Flex.Box>
  );
};
