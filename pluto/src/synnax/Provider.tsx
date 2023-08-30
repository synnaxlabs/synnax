// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  PropsWithChildren,
  ReactElement,
  createContext,
  useCallback,
  useContext,
} from "react";

import { ConnectionState, Synnax, SynnaxProps, TimeSpan } from "@synnaxlabs/client";
import { Case } from "@synnaxlabs/x";

import { Aether } from "@/aether";
import { useAsyncEffect } from "@/hooks/useAsyncEffect";
import { useCombinedStateAndRef } from "@/hooks/useCombinedStateAndRef";
import { Status } from "@/status";
import { synnax } from "@/synnax/aether";

const Context = createContext<synnax.ContextValue>(synnax.ZERO_CONTEXT_VALUE);

export const use = (): Synnax | null => useContext(Context).synnax;

export const useConnectionState = (): ConnectionState => useContext(Context).state;

export interface ProviderProps extends PropsWithChildren {
  connParams?: SynnaxProps;
}

const CONNECTION_STATE_VARIANT: Record<ConnectionState["status"], Status.Variant> = {
  connected: "success",
  connecting: "info",
  disconnected: "info",
  failed: "error",
};

export const Provider = Aether.wrap<ProviderProps>(
  synnax.Provider.TYPE,
  ({ aetherKey, connParams, children }): ReactElement => {
    const [state, setState, ref] = useCombinedStateAndRef<synnax.ContextValue>(
      synnax.ZERO_CONTEXT_VALUE
    );

    const [{ path }, , setAetherState] = Aether.use({
      aetherKey,
      type: synnax.Provider.TYPE,
      schema: synnax.Provider.stateZ,
      initialState: { props: connParams ?? null, state: null },
    });

    const add = Status.useAggregator();

    const handleChange = useCallback(
      (state: ConnectionState) => {
        if (ref.current.state.status !== state.status) {
          add({
            variant: CONNECTION_STATE_VARIANT[state.status],
            message: state.message ?? Case.capitalize(state.status),
          });
        }
        setState((prev) => ({ ...prev, state }));
      },
      [add]
    );

    useAsyncEffect(async () => {
      if (state.synnax != null) state.synnax.close();
      if (connParams == null) return setState(synnax.ZERO_CONTEXT_VALUE);

      const c = new Synnax({
        ...connParams,
        connectivityPollFrequency: TimeSpan.seconds(5),
      });

      const connectivity = await c.connectivity.check();

      setState({
        synnax: c,
        state: connectivity,
      });
      add({
        variant: CONNECTION_STATE_VARIANT[connectivity.status],
        message: connectivity.message ?? connectivity.status.toUpperCase(),
      });

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
  }
);
