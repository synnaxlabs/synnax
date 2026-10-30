// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { connection, Synnax, type SynnaxParams } from "@synnaxlabs/client";
import { context } from "@synnaxlabs/lyra/context";
import { useCombinedStateAndRef, useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { useMemoDeepEqual } from "@synnaxlabs/lyra/memo";
import { Status } from "@synnaxlabs/lyra/status";
import { migrate, record } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  useEffect,
  useRef,
  useState,
} from "react";
import z from "zod";

import { Aether } from "@/aether";
import { Errors } from "@/errors";
import { synnax } from "@/synnax/aether";

export interface ContextValue extends synnax.ContextValue {
  status: connection.Status;
}

const ZERO_CONTEXT_VALUE: ContextValue = {
  ...synnax.ZERO_CONTEXT_VALUE,
  status: connection.DEFAULT_STATUS,
};

const [Context, useContext] = context.create({
  defaultValue: ZERO_CONTEXT_VALUE,
  displayName: "Synnax.Context",
});

export const use = () => useContext().client;

export const useConnectionStatus = () => useContext().status;

export interface ProviderProps extends PropsWithChildren {
  connParams?: SynnaxParams;
}

export const SERVER_VERSION_MISMATCH = "serverVersionMismatch";
export const CLOCK_SKEW_EXCEEDED = "clockSkewExceeded";

export const statusDetailsSchema = z.object({
  type: z.string(),
  oldServer: z.boolean(),
  nodeVersion: z.string().optional(),
  clientVersion: z.string(),
});

export const clockSkewDetailsSchema = z.object({
  type: z.string(),
  clockSkew: z.number(),
});

export interface StatusDetails extends z.infer<typeof statusDetailsSchema> {}

const addClockSkewStatus = (
  addStatus: Status.Adder,
  status: connection.Status,
): void => {
  const skew = status.details.clockSkew;
  const direction = skew.valueOf() > 0n ? "ahead of" : "behind";
  addStatus<typeof clockSkewDetailsSchema>({
    variant: "warning",
    message: "Clock skew detected",
    description:
      `This machine's clock is ${direction} the Core ` +
      `by about ${skew.abs().toString()}. This can cause ` +
      `issues with time-series data. Synchronize your system clock.`,
    details: {
      type: CLOCK_SKEW_EXCEEDED,
      clockSkew: Number(skew.valueOf()),
    },
  });
};

const reasonOf = (status: connection.Status): connection.Reason | undefined =>
  status.variant === "error" ? status.details.reason : undefined;

const addVersionMismatchStatus = (
  addStatus: Status.Adder,
  status: connection.Status,
): void => {
  const oldServer =
    status.details.nodeVersion == null ||
    migrate.semVerOlder(status.details.nodeVersion, status.details.clientVersion);
  addStatus<typeof statusDetailsSchema>({
    variant: "warning",
    message: "Incompatible Core version",
    description: `Core version ${status.details.nodeVersion != null ? `${status.details.nodeVersion} ` : ""}is ${oldServer ? "older" : "newer"} than client version ${status.details.clientVersion}. Compatibility issues may arise.`,
    details: {
      type: SERVER_VERSION_MISMATCH,
      oldServer,
      nodeVersion: status.details.nodeVersion,
      clientVersion: status.details.clientVersion,
    },
  });
};

interface TestProviderProps extends PropsWithChildren {
  client: Synnax | null;
  status?: connection.Status;
}

export const TestProvider = ({
  children,
  client,
  status,
}: TestProviderProps): ReactElement => {
  const { path } = Aether.useUnidirectional({
    type: synnax.Provider.TYPE,
    schema: synnax.Provider.stateZ,
    state: { props: null },
  });
  const value = {
    ...ZERO_CONTEXT_VALUE,
    client,
    status: status ?? ZERO_CONTEXT_VALUE.status,
  };
  return (
    <Context value={value}>
      <Aether.Composite path={path}>{children}</Aether.Composite>
    </Context>
  );
};

export const Provider = ({
  children,
  connParams: connParamsProp,
}: ProviderProps): ReactElement => {
  // The client's lifetime keys on param content: a content-equal rewrite of
  // the params object (state hydration) must not tear down the connection.
  const connParams = useMemoDeepEqual(connParamsProp);
  const [state, setState, ref] =
    useCombinedStateAndRef<ContextValue>(ZERO_CONTEXT_VALUE);
  // Advances whenever the cluster may answer differently than it just did, which is
  // the cue for latched error boundaries to try again.
  const [resetKey, setResetKey] = useState(0);

  // The worker joins the page client's session instead of opening its own, so ending
  // the page's session ends the worker's access too.
  const { client: pageClient } = state;
  // for a moment after a switch, the page client still belongs to the previous Core
  const current =
    pageClient != null &&
    connParams != null &&
    pageClient.params.host === connParams.host &&
    Number(pageClient.params.port) === Number(connParams.port) &&
    pageClient.params.secure === (connParams.secure ?? false);
  const token =
    current && state.status.details.authenticated ? pageClient.auth.token : undefined;
  let workerParams: SynnaxParams | null = null;
  if (connParams != null && token != null) {
    const shared = record.omit(connParams, "username", "password", "token");
    workerParams =
      token === "" ? { ...shared, sessionCookie: true } : { ...shared, token };
  }
  const { path } = Aether.useUnidirectional({
    type: synnax.Provider.TYPE,
    schema: synnax.Provider.stateZ,
    state: { props: workerParams },
  });

  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  const versionWarned = useRef(false);

  const handleChange = (connStatus: connection.Status) => {
    const prev = ref.current.status;
    if (
      prev.variant !== connStatus.variant ||
      prev.message !== connStatus.message ||
      reasonOf(prev) !== reasonOf(connStatus)
    )
      addStatus(connStatus);
    if (connStatus.variant === "success") {
      if (connStatus.details.clockSkewExceeded)
        addClockSkewStatus(addStatus, connStatus);
      if (!connStatus.details.clientServerCompatible && !versionWarned.current) {
        versionWarned.current = true;
        addVersionMismatchStatus(addStatus, connStatus);
      }
    }
    if (connStatus.details.epoch !== prev.details.epoch) setResetKey((key) => key + 1);
    setState((prev) => ({ ...prev, status: connStatus }));
  };
  // Read through refs, so a new callback does not rebuild the client.
  const handleChangeRef = useSyncedRef(handleChange);
  const handleErrorRef = useSyncedRef(handleError);

  useEffect(() => {
    if (connParams == null) {
      setState(ZERO_CONTEXT_VALUE);
      return;
    }
    versionWarned.current = false;
    const heading = {
      message: "Failed to sync with Core",
      name: connParams.name ?? "",
    };
    const client = new Synnax({
      ...connParams,
      onInternalError: (e) => handleErrorRef.current(e, heading),
    });
    setState({ client, status: client.connection.status });
    setResetKey((key) => key + 1);
    const detach = client.connection.onChange((connStatus) =>
      handleChangeRef.current(connStatus),
    );
    return () => {
      detach();
      // Effect cleanup cannot await. The failure is logged rather than pushed
      // through handleError: a status for a client the user already left is noise.
      client.close().catch((e: unknown) => console.error("failed to close client", e));
      setState(ZERO_CONTEXT_VALUE);
    };
  }, [connParams]);

  return (
    <Context value={state}>
      <Errors.ResetProvider value={resetKey}>
        <Aether.Composite path={path}>{children}</Aether.Composite>
      </Errors.ResetProvider>
    </Context>
  );
};
