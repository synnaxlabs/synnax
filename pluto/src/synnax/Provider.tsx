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
import { caseconv, migrate, type status } from "@synnaxlabs/x";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  use as reactUse,
  useCallback,
} from "react";

import { Aether } from "@/aether";
import { useAsyncEffect, useCombinedStateAndRef } from "@/hooks";
import { Status } from "@/status";
import { synnax } from "@/synnax/aether";

export interface ContextValue extends synnax.ContextValue {
  state: connection.State;
}

const ZERO_CONTEXT_VALUE: ContextValue = {
  ...synnax.ZERO_CONTEXT_VALUE,
  state: Synnax.connectivity.DEFAULT,
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
  disconnected: "info",
  failed: "error",
};

export const SERVER_VERSION_MISMATCH = "serverVersionMismatch";

export interface StatusDetails {
  type: string;
  oldServer: boolean;
  nodeVersion?: string;
  clientVersion: string;
}

const createErrorDescription = (
  oldServer: boolean,
  clientVersion: string,
  nodeVersion?: string,
): string =>
  `Cluster version ${nodeVersion != null ? `${nodeVersion} ` : ""}is ${oldServer ? "older" : "newer"} than client version ${clientVersion}. Compatibility issues may arise.`;

export const Provider = ({ children, connParams }: ProviderProps): ReactElement => {
  const [state, setState, ref] =
    useCombinedStateAndRef<ContextValue>(ZERO_CONTEXT_VALUE);

  const { path } = Aether.useUnidirectional({
    type: synnax.Provider.TYPE,
    schema: synnax.Provider.stateZ,
    state: { props: connParams ?? null, state: null },
  });

  const addStatus = Status.useAdder<StatusDetails | null>();

  const handleChange = useCallback(
    (state: connection.State) => {
      if (ref.current.state.status !== state.status)
        addStatus({
          variant: CONNECTION_STATE_VARIANTS[state.status],
          message: state.message ?? caseconv.capitalize(state.status),
          details: null,
        });
      setState((prev) => ({ ...prev, state }));
    },
    [addStatus],
  );

  useAsyncEffect(async () => {
    if (state.client != null) state.client.close();
    if (connParams == null) return setState(ZERO_CONTEXT_VALUE);

    const c = new Synnax({
      ...connParams,
      connectivityPollFrequency: TimeSpan.seconds(2),
    });

    setState({
      client: c,
      state: {
        clusterKey: "",
        status: "connecting",
        message: "Connecting...",
        clientServerCompatible: false,
        clientVersion: c.clientVersion,
      },
    });

    const connectivity = await c.connectivity.check();

    setState({ client: c, state: connectivity });
    addStatus({
      variant: CONNECTION_STATE_VARIANTS[connectivity.status],
      message: connectivity.message ?? connectivity.status.toUpperCase(),
      details: null,
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

      addStatus({
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

    c.connectivity.onChange(handleChange);

    return () => {
      c.close();
      setState(ZERO_CONTEXT_VALUE);
    };
  }, [connParams, handleChange]);

  return (
    <Context value={state}>
      <Aether.Composite path={path}>{children}</Aether.Composite>
    </Context>
  );
};
