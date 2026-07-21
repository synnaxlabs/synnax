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
} from "react";
import z from "zod";

import { Aether } from "@/aether";
import { context } from "@/context";
import { useCombinedStateAndRef } from "@/hooks";
import { Status } from "@/status/base";
import { synnax } from "@/synnax/aether";

export interface ContextValue extends synnax.ContextValue {
  state: connection.State;
}

const ZERO_CONTEXT_VALUE: ContextValue = {
  ...synnax.ZERO_CONTEXT_VALUE,
  state: connection.DEFAULT_STATE,
};

const [Context, useContext] = context.create({
  defaultValue: ZERO_CONTEXT_VALUE,
  displayName: "Synnax.Context",
});

export const use = () => useContext().client;

export const useConnectionState = () => useContext().state;

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

const addClockSkewStatus = (addStatus: Status.Adder, state: connection.State): void => {
  const skew = state.clockSkew;
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

const addVersionMismatchStatus = (
  addStatus: Status.Adder,
  state: connection.State,
): void => {
  const oldServer =
    state.nodeVersion == null ||
    migrate.semVerOlder(state.nodeVersion, state.clientVersion);
  addStatus<typeof statusDetailsSchema>({
    variant: "warning",
    message: "Incompatible Core version",
    description: `Core version ${state.nodeVersion != null ? `${state.nodeVersion} ` : ""}is ${oldServer ? "older" : "newer"} than client version ${state.clientVersion}. Compatibility issues may arise.`,
    details: {
      type: SERVER_VERSION_MISMATCH,
      oldServer,
      nodeVersion: state.nodeVersion,
      clientVersion: state.clientVersion,
    },
  });
};

interface TestProviderProps extends PropsWithChildren {
  client: Synnax | null;
}

export const TestProvider = ({ children, client }: TestProviderProps): ReactElement => {
  const { path } = Aether.useUnidirectional({
    type: synnax.Provider.TYPE,
    schema: synnax.Provider.stateZ,
    state: { props: null, state: null },
  });
  const value = useMemo(() => ({ ...ZERO_CONTEXT_VALUE, client }), [client]);
  return (
    <Context value={value}>
      <Aether.Composite path={path}>{children}</Aether.Composite>
    </Context>
  );
};

export const Provider = ({ children, connParams }: ProviderProps): ReactElement => {
  const [state, setState, ref] =
    useCombinedStateAndRef<ContextValue>(ZERO_CONTEXT_VALUE);

  const { path } = Aether.useUnidirectional({
    type: synnax.Provider.TYPE,
    schema: synnax.Provider.stateZ,
    state: { props: connParams ?? null, state: null },
  });

  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  const versionWarned = useRef(false);

  const handleChange = useCallback(
    (connState: connection.State) => {
      const prev = ref.current.state;
      if (prev.variant !== connState.variant || prev.reason !== connState.reason)
        addStatus({ variant: connState.variant, message: connState.message });
      if (connState.variant === "success") {
        if (connState.clockSkewExceeded) addClockSkewStatus(addStatus, connState);
        if (!connState.clientServerCompatible && !versionWarned.current) {
          versionWarned.current = true;
          addVersionMismatchStatus(addStatus, connState);
        }
      }
      setState((prev) => ({ ...prev, state: connState }));
    },
    [addStatus],
  );

  useEffect(() => {
    if (ref.current.client != null) ref.current.client.close();
    if (connParams == null) {
      setState(ZERO_CONTEXT_VALUE);
      return;
    }
    versionWarned.current = false;
    const client = new Synnax({ ...connParams, onInternalError: handleError });
    setState({ client, state: client.connection.state });
    const detach = client.connection.onChange(handleChange);
    return () => {
      detach();
      client.close();
      setState(ZERO_CONTEXT_VALUE);
    };
  }, [connParams, handleChange, handleError]);

  return (
    <Context value={state}>
      <Aether.Composite path={path}>{children}</Aether.Composite>
    </Context>
  );
};
