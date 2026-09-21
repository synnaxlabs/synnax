// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/embedded/Guard.css";

import { Button, Flex, Icon, Status, Synnax } from "@synnaxlabs/pluto";
import {
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
  useEffect,
  useState,
} from "react";

import { useDiagnosticsModal } from "@/feature/embedded/Diagnostics";
import { useStatus } from "@/feature/embedded/Provider";
import { NAME } from "@/feature/embedded/useConnParams";
import { useRestart } from "@/feature/embedded/useRestart";
import { Shell } from "@/feature/shell";
import { Access } from "@/platform/access";
import { CSS } from "@/platform/css";
import { Shell as PlatformShell } from "@/platform/shell";
import { Session } from "@/session";

/**
 * Renders a splash instead of the workspace until the embedded Core runs, the session
 * settles, and the permissions are cached. Once the workspace is up, a restart of the
 * Core leaves it mounted; only a Core that stays down replaces it.
 */
export const Guard = ({ children }: PropsWithChildren): ReactNode => {
  const status = useStatus();
  const client = Synnax.use();
  const settled = Session.useSettled();
  const selected = Session.Core.useSelectIsAnySelected();
  let content: ReactNode;
  if (status.state === "failed") content = <Failed message={status.message} />;
  else if (client == null || !selected || !settled)
    content = <Starting connected={client != null && selected} />;
  else
    content = (
      <Access.PermissionsBoundary loading={<Starting connected />}>
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

// Every guarded surface reads a denial from an empty policy set, so the workspace
// cannot render before the policies land.
interface BodyProps extends PropsWithChildren {
  revealed?: boolean;
}

const Body = ({ revealed = true, children }: BodyProps): ReactElement => (
  <Shell.Frame className={CSS.B("embedded")}>
    <Flex.Box
      y
      align="center"
      justify="center"
      gap={8}
      className={CSS.cls(CSS.BE("embedded", "body"), revealed && CSS.M("revealed"))}
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

const Starting = ({ connected }: StartingProps): ReactElement => {
  // A window that opens on a Core that already runs settles before this fires, so
  // the card never flashes a spinner.
  const [revealed, setRevealed] = useState(false);
  useEffect(() => {
    const timeout = setTimeout(() => setRevealed(true), 300);
    return () => clearTimeout(timeout);
  }, []);
  return (
    <Body revealed={revealed}>
      <Status.Summary
        variant="loading"
        message={connected ? "Preparing your workspace..." : "Starting Synnax..."}
      />
    </Body>
  );
};

interface FailedProps {
  message: string;
}

const Failed = ({ message }: FailedProps): ReactElement => {
  const openDiagnostics = useDiagnosticsModal();
  const restart = useRestart();
  return (
    <Body>
      <Status.Summary
        variant="error"
        message="Synnax stopped unexpectedly"
        description={message}
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
