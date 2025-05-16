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

const Context = createContext<synnax.ContextValue>(synnax.ZERO_CONTEXT_VALUE);

const useContext = () => reactUse(Context);

export const use = () => useContext().synnax;

export const useConnectionState = () => useContext().state;

export interface ProviderProps extends PropsWithChildren {
  connParams?: SynnaxProps;
}

export const CONNECTION_STATE_VARIANTS: Record<connection.Status, status.Variant> = {
  connected: "success",
  connecting: "info",
  disconnected: "info",
  failed: "error",
};

export const SERVER_VERSION_MISMATCH = "serverVersionMismatch";

const createErrorDescription = (
  oldServer: boolean,
  clientVersion: string,
  nodeVersion?: string,
): string =>
  `Cluster version ${nodeVersion != null ? `${nodeVersion} ` : ""}is ${oldServer ? "older" : "newer"} than client version ${clientVersion}. Compatibility issues may arise.`;

export const Provider = ({ children, connParams }: ProviderProps): ReactElement => {
  const [state, setState, ref] = useCombinedStateAndRef<synnax.ContextValue>(
    synnax.ZERO_CONTEXT_VALUE,
  );

  const [{ path }, , setAetherState] = Aether.use({
    type: synnax.Provider.TYPE,
    schema: synnax.Provider.stateZ,
    initialState: { props: connParams ?? null, state: null },
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

  useAsyncEffect(async () => {
    if (state.synnax != null) state.synnax.close();
    if (connParams == null) return setState(synnax.ZERO_CONTEXT_VALUE);

    const client = new Synnax({
      ...connParams,
      connectivityPollFrequency: TimeSpan.seconds(5),
    });

    setState({
      synnax: client,
      state: {
        clusterKey: "",
        status: "connecting",
        message: "Connecting...",
        clientServerCompatible: false,
        clientVersion: client.clientVersion,
      },
    });

    const connectivity = await client.connectivity.check();

    setState({ synnax: client, state: connectivity });
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

      addStatus({
        variant: "warning",
        message: "Incompatible cluster version",
        description,
        data: {
          type: SERVER_VERSION_MISMATCH,
          oldServer,
          nodeVersion: connectivity.nodeVersion,
          clientVersion: connectivity.clientVersion,
        },
      });
    }

    client.connectivity.onChange(handleChange);

    setAetherState({ props: connParams, state: connectivity });

    return () => {
      client.close();
      setState(synnax.ZERO_CONTEXT_VALUE);
      setAetherState({ props: null, state: null });
    };
  }, [connParams, handleChange]);

  return (
    <Context value={state}>
      <Aether.Composite path={path}>{children}</Aether.Composite>
    </Context>
  );
};
