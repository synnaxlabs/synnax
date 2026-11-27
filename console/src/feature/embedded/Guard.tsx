// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/embedded/Guard.css";

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";
import { errors, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
  useEffect,
} from "react";

import { ensure } from "@/feature/embedded/apiKey";
import { useStatus } from "@/feature/embedded/Provider";
import type { Status as SupervisorStatus } from "@/feature/embedded/supervisor";
import { NAME } from "@/feature/embedded/useConnParams";
import { useDiagnosticsModal } from "@/feature/embedded/useDiagnosticsModal";
import { useRestart } from "@/feature/embedded/useRestart";
import { Shell } from "@/feature/shell";
import { Access } from "@/platform/access";
import { CSS } from "@/platform/css";
import { Shell as PlatformShell } from "@/platform/shell";
import { Session } from "@/session";

/**
 * Renders a splash instead of the workspace until the embedded Core runs, the session
 * settles, and the permissions are cached. Once the workspace is up, a restart of the
 * Core leaves it mounted; a Core that waits, stays down, or stops for an exit replaces
 * it.
 */
export const Guard = ({ children }: PropsWithChildren): ReactNode => {
  const status = useStatus();
  const client = Synnax.use();
  const settled = Session.useSettled();
  const selected = Session.Core.useSelectIsAnySelected();
  let content: ReactNode;
  if (status.state === "failed") content = <Failed {...status} />;
  else if (status.state === "waiting") content = <Waiting />;
  else if (status.state === "exiting") content = <Exiting />;
  else if (client == null || !selected || !settled)
    content = <Starting connected={client != null && selected} />;
  else
    content = (
      <Access.PermissionsBoundary loading={<Starting connected />}>
        <APIKeySideEffect />
        {children}
      </Access.PermissionsBoundary>
    );
  return (
    <>
      <SideEffect />
      {content}
    </>
  );
};

/**
 * Selects the record of the embedded Core once it runs. Stored state is split by the
 * cluster key that the session caches on the selected record.
 */
const SideEffect = (): null => {
  const status = useStatus();
  const dispatch = Session.useDispatch();
  const connection = status.state === "running" ? status.connection : null;
  const host = connection?.host;
  const port = connection?.port;
  const username = connection?.username;
  useEffect(() => {
    if (host == null || port == null || username == null) return;
    // Every window shares one store, so the main window writes for all of them.
    if (!Session.Runtime.isMainWindow()) return;
    const key = Session.Core.EMBEDDED_KEY;
    dispatch(
      Session.Core.set({
        key,
        name: NAME,
        host,
        port,
        username,
        password: "",
        secure: false,
      }),
    );
    dispatch(Session.Core.select(key));
  }, [dispatch, host, port, username]);
  return null;
};

// setTimeout fires at once for a longer delay.
const MAX_TIMEOUT = TimeSpan.milliseconds(2 ** 31 - 1);
const FIRST_RETRY = TimeSpan.seconds(10);
const MAX_RETRY_DOUBLINGS = 6;

/**
 * Keeps an API key for the scripts on this computer saved with the Core address, and
 * saves a new one when it expires.
 */
const APIKeySideEffect = (): null => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  useEffect(() => {
    if (client == null || !Session.Runtime.isMainWindow()) return;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let disposed = false;
    let failures = 0;
    const after = (span: TimeSpan): void => {
      if (disposed) return;
      const wait = span.greaterThan(MAX_TIMEOUT) ? MAX_TIMEOUT : span;
      timer = setTimeout(keep, wait.milliseconds);
    };
    // A failure reports once, then retries with backoff until a save works.
    const keep = (): void =>
      handleError(async () => {
        let expires: TimeStamp;
        try {
          ({ expires } = await ensure(client));
        } catch (e) {
          after(FIRST_RETRY.mult(2 ** Math.min(failures, MAX_RETRY_DOUBLINGS)));
          failures += 1;
          if (failures === 1) throw errors.fromUnknown(e);
          return;
        }
        failures = 0;
        if (!expires.equals(TimeStamp.MAX))
          after(TimeStamp.now().span(expires).add(TimeSpan.SECOND));
      }, "Failed to save API key for scripts");
    keep();
    return () => {
      disposed = true;
      clearTimeout(timer);
    };
  }, [client, handleError]);
  return null;
};

interface BodyProps extends PropsWithChildren {
  /** Holds the content back for a moment, so a short wait never flashes a spinner. */
  delayed?: boolean;
}

const Body = ({ delayed = false, children }: BodyProps): ReactElement => (
  <Shell.Frame className={CSS.B("embedded")}>
    <Flex.Box
      y
      align="center"
      justify="center"
      gap={8}
      className={CSS.cls(CSS.BE("embedded", "body"), delayed && CSS.M("delayed"))}
    >
      <Status.Orbital core={<PlatformShell.Mark />} />
      {children}
    </Flex.Box>
  </Shell.Frame>
);

interface StartingProps {
  /** True once a client reaches the Core and the session is left to settle. */
  connected: boolean;
}

// A window that opens on a Core that already runs settles within the delay.
const Starting = ({ connected }: StartingProps): ReactElement => (
  <Body delayed>
    <Status.Summary
      variant="loading"
      message={connected ? "Preparing your workspace..." : "Starting Synnax..."}
    />
  </Body>
);

const Waiting = (): ReactElement => (
  <Body delayed>
    <Status.Summary
      variant="loading"
      message="Waiting for Synnax to stop..."
      description="Synnax from the last session is still shutting down. It starts here when that finishes."
    />
  </Body>
);

const Exiting = (): ReactElement => (
  <Body delayed>
    <Status.Summary
      variant="loading"
      message="Stopping Synnax..."
      description="Waiting for running tasks to stop."
    />
  </Body>
);

interface FailedProps extends Pick<
  Extract<SupervisorStatus, { state: "failed" }>,
  "reason" | "message"
> {}

const Failed = ({ reason, message }: FailedProps): ReactElement => {
  const openDiagnostics = useDiagnosticsModal();
  const restart = useRestart();
  const locked = reason === "locked";
  return (
    <Body>
      <Status.Summary
        variant="error"
        message={locked ? "Data folder in use" : "Synnax stopped unexpectedly"}
        description={
          locked
            ? "Another Synnax process is still using the data folder. End it, then select Restart."
            : message
        }
      />
      <Flex.Box x gap="small">
        <Button.Button variant="outlined" onClick={() => openDiagnostics()}>
          <Icon.Hardware />
          Diagnostics
        </Button.Button>
        <Button.Button variant="filled" onClick={restart}>
          <Icon.Refresh />
          Restart
        </Button.Button>
      </Flex.Box>
    </Body>
  );
};
