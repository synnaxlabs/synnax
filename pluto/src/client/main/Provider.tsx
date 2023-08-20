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
  useContext,
  useState,
} from "react";

import { ConnectionState, Synnax, SynnaxProps, TimeSpan } from "@synnaxlabs/client";

import { Aether } from "@/aether/main";
import { client } from "@/client/aether";
import { useAsyncEffect } from "@/hooks/useAsyncEffect";

const Context = createContext<client.ContextValue>(client.ZERO_CONTEXT_VALUE);

export const use = (): Synnax | null => useContext(Context).client;

export const useConnectionState = (): ConnectionState => useContext(Context).state;

export interface ProviderProps extends PropsWithChildren {
  connParams?: SynnaxProps;
}

export const Provider = Aether.wrap<ProviderProps>(
  client.Provider.TYPE,
  ({ aetherKey, connParams, children }): ReactElement => {
    const [state, setState] = useState<client.ContextValue>(client.ZERO_CONTEXT_VALUE);

    const [{ path }, , setAetherState] = Aether.use({
      aetherKey,
      type: client.Provider.TYPE,
      schema: client.Provider.stateZ,
      initialState: { props: connParams ?? null, state: null },
    });

    useAsyncEffect(async () => {
      if (state.client != null) state.client.close();
      if (connParams == null) return setState(client.ZERO_CONTEXT_VALUE);

      const c = new Synnax({
        ...connParams,
        connectivityPollFrequency: TimeSpan.seconds(5),
      });

      const connectivity = await c.connectivity.check();

      setState({ client: c, state: connectivity });

      c.connectivity.onChange((state) => setState((prev) => ({ ...prev, state })));

      setAetherState({ props: connParams, state: connectivity });

      return () => {
        c.close();
        setState(client.ZERO_CONTEXT_VALUE);
        setAetherState({ props: null, state: null });
      };
    }, [connParams]);

    return (
      <Context.Provider value={state}>
        <Aether.Composite path={path}>{children}</Aether.Composite>
      </Context.Provider>
    );
  }
);
