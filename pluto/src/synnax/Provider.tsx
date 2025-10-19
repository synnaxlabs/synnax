// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type connection,
  Synnax,
  type SynnaxProps,
  TimeSpan,
} from "@synnaxlabs/client";
import { type breaker, caseconv, migrate, type status } from "@synnaxlabs/x";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  use as reactUse,
  useCallback,
  useMemo,
} from "react";
import z from "zod";

import { Aether } from "@/aether";
import { useAsyncEffect, useCombinedStateAndRef } from "@/hooks";
import { Status } from "@/status/core";
import { synnax } from "@/synnax/aether";

export interface ContextValue extends synnax.ContextValue {
  state: connection.State;
}

const ZERO_CONTEXT_VALUE: ContextValue = {
  ...synnax.ZERO_CONTEXT_VALUE,
  state: Synnax.connectivity.DEFAULT,
};

const DEFAULT_RETRY_CONFIG: breaker.Config = {
  maxRetries: 4,
  baseInterval: TimeSpan.seconds(1),
  scale: 2,
};

const Context = createContext<ContextValue>(ZERO_CONTEXT_VALUE);

const useContext = () => reactUse(Context);

export const use = () => useContext().client;

export const useConnectionState = () => useContext().state;

export interface ProviderProps extends PropsWithChildren {
  connParams?: SynnaxProps;
}

export const CONNECTION_STATE_VARIANTS: Record<connection.Status, status.Variant> = {
  connected: "success",
  connecting: "loading",
  disconnected: "disabled",
  failed: "error",
};

export const SERVER_VERSION_MISMATCH = "serverVersionMismatch";

export const statusDetailsSchema = z.object({
  type: z.string(),
  oldServer: z.boolean(),
  nodeVersion: z.string().optional(),
  clientVersion: z.string(),
});

export interface StatusDetails extends z.infer<typeof statusDetailsSchema> {}

const createErrorDescription = (
  oldServer: boolean,
  clientVersion: string,
  nodeVersion?: string,
): string =>
  `Cluster version ${nodeVersion != null ? `${nodeVersion} ` : ""}is ${oldServer ? "older" : "newer"} than client version ${clientVersion}. Compatibility issues may arise.`;

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

  const handleChange = useCallback(
    (state: connection.State) => {
      if (ref.current.state.status !== state.status)
        addStatus({
          variant: CONNECTION_STATE_VARIANTS[state.status],
          message: state.message ?? caseconv.capitalize(state.status),
        });
      setState((prev) => ({ ...prev, state }));
    },
    [addStatus],
  );

  useAsyncEffect(
    async (signal) => {
      if (state.client != null) state.client.close();
      if (connParams == null) return setState(ZERO_CONTEXT_VALUE);

      const client = new Synnax({
        retry: DEFAULT_RETRY_CONFIG,
        ...connParams,
        connectivityPollFrequency: TimeSpan.seconds(2),
      });

      setState({
        client,
        state: {
          clusterKey: "",
          status: "connecting",
          message: "Connecting...",
          clientServerCompatible: false,
          clientVersion: client.clientVersion,
        },
      });

      const connectivity = await client.connectivity.check();
      if (signal.aborted) return;

      setState({ client, state: connectivity });
      addStatus({
        variant: CONNECTION_STATE_VARIANTS[connectivity.status],
        message: connectivity.message ?? connectivity.status.toUpperCase(),
      });

      if (connectivity.status === "connected" && !connectivity.clientServerCompatible) {
        const oldServer =
          connectivity.nodeVersion == null ||
          migrate.semVerOlder(connectivity.nodeVersion, connectivity.clientVersion);

        const description = createErrorDescription(
          oldServer,
          connectivity.clientVersion,
          connectivity.nodeVersion,
        );

        addStatus<StatusDetails>({
          variant: "warning",
          message: "Incompatible cluster version",
          description,
          details: {
            type: SERVER_VERSION_MISMATCH,
            oldServer,
            nodeVersion: connectivity.nodeVersion,
            clientVersion: connectivity.clientVersion,
          },
        });
      }

      client.connectivity.onChange(handleChange);

      return () => {
        client.close();
        setState(ZERO_CONTEXT_VALUE);
      };
    },
    [connParams, handleChange],
  );

  return (
    <Context value={state}>
      <Aether.Composite path={path}>{children}</Aether.Composite>
    </Context>
  );
};
