// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { connection, Synnax, type SynnaxParams } from "@synnaxlabs/client";
import { migrate } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import z from "zod";

import { Aether } from "@/aether";
import { context } from "@/context";
import { Errors } from "@/errors";
import { useCombinedStateAndRef } from "@/hooks";
import { useMemoDeepEqual } from "@/memo";
import { Status } from "@/status/base";
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
      `This machine's clock is ${direction} the Synnax core ` +
      `by approximately ${skew.abs().toString()}. This may cause ` +
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
  const value = useMemo(
    () => ({
      ...ZERO_CONTEXT_VALUE,
      client,
      status: status ?? ZERO_CONTEXT_VALUE.status,
    }),
    [client, status],
  );
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

  const { path } = Aether.useUnidirectional({
    type: synnax.Provider.TYPE,
    schema: synnax.Provider.stateZ,
    state: { props: connParams ?? null },
  });

  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  const versionWarned = useRef(false);

  const handleChange = useCallback(
    (connStatus: connection.Status) => {
      const prev = ref.current.status;
      if (
        prev.variant !== connStatus.variant ||
        prev.message !== connStatus.message ||
        reasonOf(prev) !== reasonOf(connStatus)
      )
        addStatus({ variant: connStatus.variant, message: connStatus.message });
      if (connStatus.variant === "success") {
        if (connStatus.details.clockSkewExceeded)
          addClockSkewStatus(addStatus, connStatus);
        if (!connStatus.details.clientServerCompatible && !versionWarned.current) {
          versionWarned.current = true;
          addVersionMismatchStatus(addStatus, connStatus);
        }
      }
      if (connStatus.details.epoch !== prev.details.epoch)
        setResetKey((key) => key + 1);
      setState((prev) => ({ ...prev, status: connStatus }));
    },
    [addStatus],
  );

  useEffect(() => {
    if (connParams == null) {
      setState(ZERO_CONTEXT_VALUE);
      return;
    }
    versionWarned.current = false;
    const client = new Synnax({ ...connParams, onInternalError: handleError });
    setState({ client, status: client.connection.status });
    setResetKey((key) => key + 1);
    const detach = client.connection.onChange(handleChange);
    return () => {
      detach();
      client.close();
      setState(ZERO_CONTEXT_VALUE);
    };
  }, [connParams, handleChange, handleError]);

  return (
    <Context value={state}>
      <Errors.ResetProvider value={resetKey}>
        <Aether.Composite path={path}>{children}</Aether.Composite>
      </Errors.ResetProvider>
    </Context>
  );
};
