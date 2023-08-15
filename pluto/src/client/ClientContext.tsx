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

import { AetherClient } from "@/client/aether";
import { Aether } from "@/core/aether/main";
import { useAsyncEffect } from "@/core/hooks/useAsyncEffect";

const ClientContext = createContext<AetherClient.ContextValue>(
  AetherClient.ZERO_CONTEXT_VALUE
);

export const useClient = (): Synnax | null => useContext(ClientContext).client;

export const useConnectionState = (): ConnectionState =>
  useContext(ClientContext).state;

export interface ClientProviderProps extends PropsWithChildren {
  connParams?: SynnaxProps;
}

export const ClientProvider = Aether.wrap<ClientProviderProps>(
  AetherClient.Provider.TYPE,
  ({ aetherKey, connParams, children }): ReactElement => {
    const [state, setState] = useState<AetherClient.ContextValue>(
      AetherClient.ZERO_CONTEXT_VALUE
    );

    const [{ path }, , setAetherState] = Aether.use({
      aetherKey,
      type: AetherClient.Provider.TYPE,
      schema: AetherClient.Provider.stateZ,
      initialState: { props: connParams ?? null, state: null },
    });

    useAsyncEffect(async () => {
      if (state.client != null) state.client.close();
      if (connParams == null) return setState(AetherClient.ZERO_CONTEXT_VALUE);

      const client = new Synnax({
        ...connParams,
        connectivityPollFrequency: TimeSpan.seconds(5),
      });

      const connectivity = await client.connectivity.check();

      setState({ client, state: connectivity });

      client.connectivity.onChange((state) => setState((prev) => ({ ...prev, state })));

      setAetherState({ props: connParams, state: connectivity });

      return () => {
        client.close();
        setState(AetherClient.ZERO_CONTEXT_VALUE);
        setAetherState({ props: null, state: null });
      };
    }, [connParams]);

    return (
      <ClientContext.Provider value={state}>
        <Aether.Composite path={path}>{children}</Aether.Composite>
      </ClientContext.Provider>
    );
  }
);
