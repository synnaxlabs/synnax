// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { connection, Synnax, type SynnaxProps, TimeSpan } from "@synnaxlabs/client";
import { caseconv, migrate } from "@synnaxlabs/x";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useContext,
} from "react";

import { Aether } from "@/aether";
import { useAsyncEffect, useCombinedStateAndRef } from "@/hooks";
import { Status } from "@/status";
import { synnax } from "@/synnax/aether";

const Context = createContext<synnax.ContextValue>(synnax.ZERO_CONTEXT_VALUE);

export const use = (): Synnax | null => useContext(Context).synnax;

export const useConnectionState = (): connection.State => useContext(Context).state;

export interface ProviderProps extends PropsWithChildren {
  connParams?: SynnaxProps;
}

const CONNECTION_STATE_VARIANT: Record<connection.Status, Status.Variant> = {
  connected: "success",
  connecting: "info",
  disconnected: "info",
  failed: "error",
};

export const Provider = Aether.wrap<ProviderProps>(
  synnax.Provider.TYPE,
  ({ aetherKey, connParams, children }): ReactElement => {
    const [state, setState, ref] = useCombinedStateAndRef<synnax.ContextValue>(
      synnax.ZERO_CONTEXT_VALUE,
    );

    const [{ path }, , setAetherState] = Aether.use({
      aetherKey,
      type: synnax.Provider.TYPE,
      schema: synnax.Provider.stateZ,
      initialState: { props: connParams ?? null, state: null },
    });

    const add = Status.useAggregator();

    const handleChange = useCallback(
      (state: connection.State) => {
        if (ref.current.state.status !== state.status) {
          add({
            variant: CONNECTION_STATE_VARIANT[state.status],
            message: state.message ?? caseconv.capitalize(state.status),
          });
        }
        setState((prev) => ({ ...prev, state }));
      },
      [add],
    );

    useAsyncEffect(async () => {
      if (state.synnax != null) state.synnax.close();
      if (connParams == null) return setState(synnax.ZERO_CONTEXT_VALUE);

      const c = new Synnax({
        ...connParams,
        connectivityPollFrequency: TimeSpan.seconds(5),
      });

      setState({
        synnax: c,
        state: {
          clusterKey: "",
          status: "connecting",
          message: "Connecting...",
          clientServerCompatible: false,
          clientVersion: c.clientVersion,
        },
      });

      const connectivity = await c.connectivity.check();

      setState({ synnax: c, state: connectivity });
      add({
        variant: CONNECTION_STATE_VARIANT[connectivity.status],
        message: connectivity.message ?? connectivity.status.toUpperCase(),
      });

      if (connectivity.status === "connected" && !connectivity.clientServerCompatible) {
        const oldServer =
          connectivity.nodeVersion == null ||
          migrate.semVerOlder(connectivity.nodeVersion, connectivity.clientVersion);

        let description: string;
        if (!oldServer)
          description = `
        Cluster version ${connectivity.nodeVersion} is newer than client version ${connectivity.clientVersion}.
        Compatibility issues may arise. 
        `;
        else if (connectivity.nodeVersion != null)
          description = `Cluster version ${connectivity.nodeVersion} is older than client version ${connectivity.clientVersion}.
        Compatibility issues may arise. 
        `;
        else
          description = `Cluster version is older than client version ${connectivity.clientVersion}. Compatibility issues may arise. `;

        add({
          variant: "warning",
          message: "Incompatible cluster version",
          description,
          data: {
            type: "serverVersionMismatch",
            oldServer,
            nodeVersion: connectivity.nodeVersion,
            clientVersion: connectivity.clientVersion,
          },
        });
      }

      c.connectivity.onChange(handleChange);

      setAetherState({ props: connParams, state: connectivity });

      return () => {
        c.close();
        setState(synnax.ZERO_CONTEXT_VALUE);
        setAetherState({ props: null, state: null });
      };
    }, [connParams, handleChange]);

    return (
      <Context.Provider value={state}>
        <Aether.Composite path={path}>{children}</Aether.Composite>
      </Context.Provider>
    );
  },
);
